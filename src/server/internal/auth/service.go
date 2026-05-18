package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Login(ctx context.Context, username, password, deviceName string) (string, error)
	AuthenticateToken(ctx context.Context, rawToken string) (int, error)
	Logout(ctx context.Context, rawToken string) error
}

type service struct {
	repo Repository
	log  *zap.Logger
}

func NewService(repo Repository, log *zap.Logger) Service {
	return &service{repo: repo, log: log}
}

func (s *service) AuthenticateToken(ctx context.Context, rawToken string) (int, error) {
	tokenHash := s.hashToken(rawToken)

	userID, err := s.repo.FindUserIDByHash(ctx, tokenHash)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (s *service) Login(ctx context.Context, username string, password string, deviceName string) (string, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)

	if err != nil {
		s.log.Warn("Login failed: user not found", zap.String("username", username))
		return "", errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(strings.TrimSpace(password))); err != nil {
		s.log.Warn("Login failed: incorrect password", zap.String("username", username))
		return "", errors.New("invalid username or password")
	}

	rawToken, err := s.generateSecureRandomString(32)
	if err != nil {
		s.log.Error("Failed to generate secure token", zap.Error(err))
		return "", errors.New("internal server error")
	}
	token := "oblak_pat_" + rawToken
	tokenHash := s.hashToken(token)

	key := &APIKey{
		UserID:    user.ID,
		KeyHash:   tokenHash,
		Name:      deviceName,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.repo.CreateAPIKey(ctx, key); err != nil {
		s.log.Error("Failed to save API key hash to database", zap.Error(err))
		return "", errors.New("internal server error")
	}

	s.log.Info("User successfully logged in via CLI", zap.String("username", username), zap.Int("user_id", user.ID))
	return token, nil
}

func (s *service) Logout(ctx context.Context, rawToken string) error {
	tokenHash := s.hashToken(rawToken)
	return s.repo.DeleteAPIKeyByHash(ctx, tokenHash)
}

func (s *service) generateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *service) hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
