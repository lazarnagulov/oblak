package deployment

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

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
	log         *zap.Logger
}

func NewHandler(service Service, authService auth.Service, rateLimit limiter.LimitHandler, log *zap.Logger) *Handler {
	return &Handler{
		service:     service,
		authService: authService,
		rateLimit:   rateLimit,
		log:         log,
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

	err = h.service.Deploy(c.Request.Context(), userID, manifest, fileContent)
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
	c.JSON(http.StatusCreated, gin.H{"message": "Function deployed successfully"})
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
