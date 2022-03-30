package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Logger *zap.Logger

const (
	maxSize    = 50 // the maximum size in megabytes of the log file
	maxBackups = 30 // the maximum number of old log files to retain
	maxAge     = 28 // the maximum number of days to retain old log files based on the timestamp encoded in their filename
)

func SetupLogger(logFile, mode string) (logger *zap.Logger, err error) {
	if mode == "release" {
		logger, err = NewProductionLogger(logFile)
	} else {
		logger, err = NewDevelopmentLogger(logFile)
	}
	Logger = logger
	return
}

func rotateWriteSyncer(logFile string) zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize, // megabytes
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
	})
}

func NewProductionLogger(logFile string) (*zap.Logger, error) {
	c := zap.NewProductionConfig()

	c.DisableCaller = true
	c.DisableStacktrace = true

	return c.Build(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		w := rotateWriteSyncer(logFile)
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			w,
			zap.InfoLevel,
		)
		return zapcore.NewTee(c, core)
	}))
}

func NewDevelopmentLogger(logFile string) (*zap.Logger, error) {
	c := zap.NewDevelopmentConfig()

	return c.Build(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		w := rotateWriteSyncer(logFile)
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
			w,
			zap.DebugLevel,
		)
		return zapcore.NewTee(c, core)
	}))
}
