package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lazarnagulov/oblak/server/internal/platform/logger"
	"go.uber.org/zap"
)

func main() {
	log, err := logger.New("production")
	if err != nil {
		log.Panic("Failed to create logger")
	}
	r := NewRouter(log).Build()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	log.Info("starting server",
		zap.String("port", "8080"),
		zap.String("environment", "production"),
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}
}
