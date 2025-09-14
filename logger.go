package supago

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewOpinionatedLogger creates a new logger (stylistically opinionated)
func NewOpinionatedLogger(LogLevel zapcore.Level, LogJsonFmt bool) *zap.SugaredLogger {
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
		panic(fmt.Sprintf("failed to create Logger: %v", err))
	} else {
		return lgr.Sugar()
	}
}
