package deployment

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/auth"
	"go.uber.org/zap"
)

type Handler struct {
	service     Service
	authService auth.Service
	log         *zap.Logger
}

func NewHandler(service Service, authService auth.Service, log *zap.Logger) *Handler {
	return &Handler{service: service, authService: authService, log: log}
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

	err = h.service.Deploy(c.Request.Context(), userID, manifest, fileContent)
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
	c.JSON(http.StatusCreated, gin.H{"message": "Function deployed successfully"})
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
