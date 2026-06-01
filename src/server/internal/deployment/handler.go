package deployment

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/auth"
	"github.com/lazarnagulov/oblak/server/internal/platform/httputil"
	"github.com/lazarnagulov/oblak/server/internal/platform/limiter"
	"go.uber.org/zap"
)

type Handler struct {
	service     Service
	authService auth.Service
	rateLimit   limiter.LimitHandler
	apiURL      string
	log         *zap.Logger
}

func NewHandler(service Service, authService auth.Service, rateLimit limiter.LimitHandler, apiURL string, log *zap.Logger) *Handler {
	return &Handler{
		service:     service,
		authService: authService,
		rateLimit:   rateLimit,
		log:         log,
		apiURL:      apiURL,
	}
}

// Deploy uploads a new function artifact, verifies and registers it.
// @Summary Deploy a new function
// @Description Uploads a function artifact (ZIP), verifies and registers it in the system.
// @Tags functions
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Name of the function"
// @Param runtime formData string true "Runtime environment (e.g., python3.10)"
// @Param module formData string true "Module name"
// @Param handler formData string true "Handler entrypoint (e.g., main.handler)"
// @Param timeout formData int true "Execution timeout in seconds (1-30)"
// @Param memory formData int true "Allocated memory in MB (64-512)"
// @Param file formData file true "Function ZIP artifact"
// @Success 200 {object} FunctionResponse "Function deployed successfully"
// @Failure 400 {object} httputil.ErrorResponse "Invalid request or file"
// @Failure 401 {object} httputil.ErrorResponse "Unauthorized"
// @Failure 429 {object} httputil.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} httputil.ErrorResponse "Internal server error"
// @Router /functions/ [post]
func (h *Handler) Deploy(c *gin.Context) {
	userID := c.GetInt("userID")
	mainfestJSON := c.PostForm("manifest")

	var manifest DeployRequestManifest
	if err := json.Unmarshal([]byte(mainfestJSON), &manifest); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "Invalid manifest format")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 50<<20)
	file, err := c.FormFile("artifact")
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "Artifact zip file is required or too large")
		return
	}

	fileContent, err := file.Open()
	if err != nil {
		httputil.WriteError(c, http.StatusInternalServerError, "Failed to read file")
		return
	}
	defer fileContent.Close()

	accessToken, err := h.service.Deploy(c.Request.Context(), userID, manifest, fileContent)
	if err != nil {
		if errors.Is(err, ErrFunctionAlreadyExists) {
			httputil.WriteError(c, http.StatusConflict, err.Error())
			return
		}
		var verifyErr *ErrVerificationFailed
		if errors.As(err, &verifyErr) {
			httputil.WriteError(c, http.StatusUnprocessableEntity, verifyErr.Error())
			return
		}
		h.log.Error("Service deployment failed", zap.Error(err))
		httputil.WriteError(c, http.StatusInternalServerError, "Deployment processing failed")
		return
	}

	accessURL := fmt.Sprintf("%s/execute/%s", h.apiURL, accessToken)

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Function deployed successfully",
		"access_url": accessURL,
	})
}

// List retrieves all deployed functions for the authenticated user.
// @Summary List functions
// @Description Retrieves a list of all deployed functions belonging to the authenticated user.
// @Tags functions
// @Security BearerAuth
// @Produce json
// @Success 200 {array} deployment.FunctionListResponse "List of functions"
// @Failure 401 {object} httputil.ErrorResponse "Unauthorized"
// @Failure 429 {object} httputil.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} httputil.ErrorResponse "Internal server error"
// @Router /functions/ [get]
func (h *Handler) List(c *gin.Context) {
	userID := c.GetInt("userID")
	funcs, err := h.service.ListByUserID(c.Request.Context(), userID)
	if err != nil {
		httputil.WriteError(c, http.StatusInternalServerError, "Failed to fetch functions")
		return
	}

	var response []FunctionListResponse
	for _, f := range funcs {
		response = append(response, ToListResponse(&f))
	}

	c.JSON(http.StatusOK, gin.H{"functions": response})
}

// Describe retrieves detailed information about a specific function.
// @Summary Get function details
// @Description Retrieves detailed information about a specific function by its name.
// @Tags functions
// @Security BearerAuth
// @Produce json
// @Param name path string true "Function name"
// @Success 200 {object} FunctionResponse "Function details"
// @Failure 401 {object} httputil.ErrorResponse "Unauthorized"
// @Failure 404 {object} httputil.ErrorResponse "Function not found"
// @Failure 429 {object} httputil.ErrorResponse "Rate limit exceeded"
// @Router /functions/{name} [get]
func (h *Handler) Describe(c *gin.Context) {
	name := c.Param("name")
	userID := c.GetInt("userID")

	f, err := h.service.GetByName(c.Request.Context(), userID, name)
	if err != nil {
		httputil.WriteError(c, http.StatusNotFound, "Function not found")
		return
	}

	c.JSON(http.StatusOK, ToResponse(f))
}

// Delete completely removes a function and its artifact.
// @Summary Delete a function
// @Description Deletes a function from the database and removes its artifact from the storage.
// @Tags functions
// @Security BearerAuth
// @Produce json
// @Param name path string true "Function name"
// @Success 200 {object} httputil.ErrorResponse "Success message"
// @Failure 401 {object} httputil.ErrorResponse "Unauthorized"
// @Failure 404 {object} httputil.ErrorResponse "Function not found"
// @Failure 429 {object} httputil.ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} httputil.ErrorResponse "Failed to delete artifact or database record"
// @Router /functions/{name} [delete]
func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("name")
	userID := c.GetInt("userID")

	err := h.service.Delete(c.Request.Context(), userID, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httputil.WriteError(c, http.StatusNotFound, "Function not found")
			return
		}
		httputil.WriteError(c, http.StatusInternalServerError, "Failed to delete function")
		return
	}

	c.JSON(http.StatusNoContent, gin.H{"message": "Function deleted successfully"})
}

func (h *Handler) ExecuteByTokenBody(c *gin.Context) {
	var payload []byte = nil
	if c.Request.Body != nil {
		var err error
		payload, err = io.ReadAll(c.Request.Body)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
	}
	h.executeByToken(c, payload)
}

func (h *Handler) ExecuteByTokenQuery(c *gin.Context) {
	payloadMap := make(map[string]any)

	for key, values := range c.Request.URL.Query() {
		if len(values) == 1 {
			payloadMap[key] = values[0]
		} else {
			payloadMap[key] = values
		}
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process query parameters"})
		return
	}
	h.executeByToken(c, payloadBytes)
}

func (h *Handler) executeByToken(c *gin.Context, payload []byte) {
	token := c.Param("token")

	result, err := h.service.ExecuteByAccessToken(c.Request.Context(), token, payload)
	if err != nil {
		if errors.Is(err, ErrAccessTokenInvalid) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Access token invalid or expired"})
			return
		}
		h.log.Error("Execution failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Execution failed"})
		return
	}

	c.JSON(http.StatusOK, ExecuteResponse{Status: result.Status, Logs: result.Logs, ErrorMessage: result.ErrorMessage, Result: result.Result, ExecutionTimeMs: result.ExecutionTimeMs})
}

func (h *Handler) GenerateURL(c *gin.Context) {
	name := c.Param("name")
	userID := c.GetInt("userID")

	accessToken, err := h.service.GenerateAccessToken(c.Request.Context(), userID, name)
	if err != nil {
		if errors.Is(err, ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	accessURL := fmt.Sprintf("%s/execute/%s", h.apiURL, accessToken)
	c.JSON(http.StatusOK, gin.H{"access_url": accessURL})
}

func (h *Handler) Invoke(c *gin.Context) {
	name := c.Param("name")
	userID := c.GetInt("userID")

	var body []byte
	var payload []byte = nil
	var async bool
	if c.Request.Body != nil {
		var err error
		body, err = io.ReadAll(c.Request.Body)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		if len(body) > 0 {
			jsonBody := make(map[string]any)
			if err := json.Unmarshal(body, &jsonBody); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Request body must be valid JSON"})
				return
			}
			payloadValue, exists := jsonBody["payload"]
			if exists {
				payloadBytes, err := json.Marshal(payloadValue)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process payload"})
					return
				}
				payload = payloadBytes
			}
			async = jsonBody["async"] == true
		}
	}

	if async {
		go h.service.Invoke(c.Request.Context(), userID, name, payload)
		c.JSON(http.StatusAccepted, gin.H{"message": "Function invoked asynchronously"})
		return
	}

	result, err := h.service.Invoke(c.Request.Context(), userID, name, payload)
	if err != nil {
		if errors.Is(err, ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invoke function"})
		return
	}

	c.JSON(http.StatusOK, ExecuteResponse{ID: result.ID, Status: result.Status, Logs: result.Logs, ErrorMessage: result.ErrorMessage, Result: result.Result, ExecutionTimeMs: result.ExecutionTimeMs, StartedAt: result.StartedAt, FinishedAt: result.FinishedAt})
}

func (h *Handler) ListExecutions(c *gin.Context) {
	name := c.Param("name")
	userID := c.GetInt("userID")

	executions, err := h.service.ListExecutions(c.Request.Context(), userID, name)
	if err != nil {
		if errors.Is(err, ErrFunctionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch execution records"})
		return
	}

	var response []ExecutionRecordSummary
	for _, record := range executions {
		response = append(response, ToExecutionRecordSummary(&record))
	}

	c.JSON(http.StatusOK, gin.H{"executions": response})
}

func (h *Handler) DescribeExecution(c *gin.Context) {
	userID := c.GetInt("userID")

	executionIDStr := c.Param("execution_id")
	executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}
	h.log.Info("Fetching execution record", zap.Int64("executionID", executionID), zap.Int("userID", userID))

	record, err := h.service.DescribeExecution(c.Request.Context(), executionID, userID)
	if err != nil {
		h.log.Error("Failed to fetch execution record", zap.Int64("executionID", executionID), zap.Int("userID", userID), zap.Error(err))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Execution record not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch execution record"})
		return
	}

	c.JSON(http.StatusOK, ToExecutionRecordResponse(record))
}
