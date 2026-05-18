package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	log *zap.Logger
}

func NewHandler(log *zap.Logger) *Handler {
	return &Handler{log: log}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// TODO: check user existance
	token, err := GeneratePAT()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	tokenHash := HashToken(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// TODO: write into db
	_ = tokenHash
	_ = expiresAt

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"username":   req.Username,
		"expires_at": expiresAt.Format(time.RFC3339),
	})

}

func GeneratePAT() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "oblak_pat_" + hex.EncodeToString(bytes), nil
}

func HashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
