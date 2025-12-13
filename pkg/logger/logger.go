package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init 初始化日誌系統
func Init(level string) error {
	var zapLevel zapcore.Level

	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	var err error

	log, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

// Sync 同步日誌
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

// Debug 除錯級別日誌
func Debug(msg string, fields ...any) {
	log.Sugar().Debugw(msg, fields...)
}

// Info 資訊級別日誌
func Info(msg string, fields ...any) {
	log.Sugar().Infow(msg, fields...)
}

// Warn 警告級別日誌
func Warn(msg string, fields ...any) {
	log.Sugar().Warnw(msg, fields...)
}

// Error 錯誤級別日誌
func Error(msg string, fields ...any) {
	log.Sugar().Errorw(msg, fields...)
}

// Fatal 致命錯誤級別日誌
func Fatal(msg string, fields ...any) {
	log.Sugar().Fatalw(msg, fields...)
}
