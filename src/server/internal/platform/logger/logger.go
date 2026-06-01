package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func resolveLogPath(env string) (string, error) {
	if env == "production" {
		return "/var/log/oblak/server/server.log", nil
	}

	root := os.Getenv("APP_ROOT")
	if root == "" {
		return "", fmt.Errorf("APP_ROOT env variable not set")
	}

	logPath := filepath.Join(root, "src", "logs", "server", "server.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	return logPath, nil
}

func New(env string) (*zap.Logger, error) {
	logPath, err := resolveLogPath(env)
	if err != nil {
		return nil, err
	}

	fileCfg := zap.NewDevelopmentEncoderConfig()
	fileCfg.TimeKey = "timestamp"
	fileCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	fileCfg.CallerKey = ""

	consoleCfg := zap.NewDevelopmentEncoderConfig()
	consoleCfg.TimeKey = "timestamp"
	consoleCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleCfg.CallerKey = ""
	if env == "production" {
		consoleCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		consoleCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	level := zap.InfoLevel
	if env != "production" {
		level = zap.DebugLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleCfg),
			zapcore.AddSync(os.Stdout),
			level,
		),
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(fileCfg),
			zapcore.AddSync(&lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    10,
				MaxBackups: 5,
				MaxAge:     30,
				Compress:   true,
			}),
			level,
		),
	)

	return zap.New(core), nil
}
