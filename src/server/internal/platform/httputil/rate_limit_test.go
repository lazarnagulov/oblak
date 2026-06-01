package httputil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lazarnagulov/oblak/server/internal/platform/httputil"
	"github.com/lazarnagulov/oblak/server/internal/platform/limiter"
	"go.uber.org/zap"
)

func TestRateLimitMiddleware_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()
	l := limiter.NewInMemoryLimiter()

	limits := map[string]limiter.RateLimiterConfig{
		"test_route": {Capacity: 2, Refill: 0.1},
	}

	rateLimit := httputil.NewRateLimiterHandler(l, limits, logger)

	r := gin.New()
	r.GET("/test", rateLimit("test_route"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	w1 := performGetRequest(r, "/test")
	if w1.Code != http.StatusOK {
		t.Errorf("expected status 200 on first request, got %d", w1.Code)
	}

	w2 := performGetRequest(r, "/test")
	if w2.Code != http.StatusOK {
		t.Errorf("expected status 200 on second request, got %d", w2.Code)
	}

	w3 := performGetRequest(r, "/test")
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429 on third request, got %d", w3.Code)
	}

	expectedBody := `{"error":"Rate limit exceeded for this action"}`
	if w3.Body.String() != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, w3.Body.String())
	}
}

func performGetRequest(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", path, nil)
	req.RemoteAddr = "127.0.0.1:1234"

	r.ServeHTTP(w, req)
	return w
}
