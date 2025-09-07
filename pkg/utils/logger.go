package utils

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func Logger() *zap.SugaredLogger {
	return logger
}

// OverrideLogger replace the logger used by supago with a custom logger
// To disable logging, use zap.NewNop().Sugar() or the DisableLogger() method
func OverrideLogger(lgr *zap.SugaredLogger) {
	logger = lgr
}

func DisableLogger() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}

func NewLogger(LogLevel zapcore.Level, LogJsonFmt bool) (*zap.SugaredLogger, error) {
	// defaults
	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(LogLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "Logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder, // color when console
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// json overrides
	if !LogJsonFmt {
		cfg.Encoding = "console"

		// use color logs
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		// disable caller filepath
		cfg.EncoderConfig.EncodeCaller = nil
		cfg.EncoderConfig.CallerKey = ""
		cfg.DisableCaller = true

		// disable stack trace
		cfg.EncoderConfig.StacktraceKey = ""
		cfg.DisableStacktrace = true
	}

	lgr, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create Logger: %v", err)
	} else {
		return lgr.Sugar(), nil
	}
}

func init() {
	lgr, err := NewLogger(zapcore.DebugLevel, false)
	if err != nil {
		panic(err)
	}
	logger = lgr
}
