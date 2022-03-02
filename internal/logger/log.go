package logger

import (
	"errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type zapLoggerString string

const (
	Z zapLoggerString = "z"
)

func New(logLevel int) (sugarLogger *zap.SugaredLogger, err error) {
	var level zapcore.Level
	switch logLevel {
	case 0:
		level = zap.DebugLevel
	case 1:
		level = zapcore.InfoLevel
	default:
		return nil, errors.New("поддерживаются уровни логгирования 0 - debug и 1 - info")
	}

	file, err := os.OpenFile("./log_file.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}

	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewConsoleEncoder(cfg), zapcore.AddSync(os.Stdout), level),
		zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(file), level),
	)

	logger := zap.New(core)
	sugarLogger = logger.Sugar()

	return sugarLogger, nil
}
