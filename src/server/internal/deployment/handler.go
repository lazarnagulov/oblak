package deployment

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/auth"
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

func (h *Handler) Deploy(c *gin.Context) {
	userID := c.GetInt("userID")
	mainfestJSON := c.PostForm("manifest")

	var manifest DeployRequestManifest
	if err := json.Unmarshal([]byte(mainfestJSON), &manifest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid manifest format"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 50<<20)
	file, err := c.FormFile("artifact")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Artifact zip file is required or too large"})
		return
	}

	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer fileContent.Close()

	accessToken, err := h.service.Deploy(c.Request.Context(), userID, manifest, fileContent)
	if err != nil {
		if errors.Is(err, ErrFunctionAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		var verifyErr *ErrVerificationFailed
		if errors.As(err, &verifyErr) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": verifyErr.Error()})
			return
		}
		h.log.Error("Service deployment failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Deployment processing failed"})
		return
	}

	accessURL := fmt.Sprintf("%s/execute/%s", h.apiURL, accessToken)

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Function deployed successfully",
		"access_url": accessURL,
	})
}

func (h *Handler) List(c *gin.Context) {
	userID := c.GetInt("userID")
	funcs, err := h.service.ListByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch functions"})
		return
	}

	var response []FunctionListResponse
	for _, f := range funcs {
		response = append(response, ToListResponse(&f))
	}

	c.JSON(http.StatusOK, gin.H{"functions": response})
}

func (h *Handler) Describe(c *gin.Context) {
	name := c.Param("name")
	userID := c.GetInt("userID")

	f, err := h.service.GetByName(c.Request.Context(), userID, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
		return
	}

	c.JSON(http.StatusOK, ToResponse(f))
}

func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("name")
	userID := c.GetInt("userID")

	err := h.service.Delete(c.Request.Context(), userID, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Function not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete function"})
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

	c.JSON(http.StatusOK, ExecuteResponse{Success: result.Success, Logs: result.Logs, ErrorMessage: result.ErrorMessage, Result: result.Result, ExecutionTimeMs: result.ExecutionTimeMs})
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

	c.JSON(http.StatusOK, ExecuteResponse{Success: result.Success, Logs: result.Logs, ErrorMessage: result.ErrorMessage, Result: result.Result, ExecutionTimeMs: result.ExecutionTimeMs})
}
