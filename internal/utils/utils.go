package utils

import (
	"go.uber.org/zap"
)

// HandleFatalError обрабатывает ошибку и вызывает panic, если она не nil.
// Если логгер nil, вызывает panic с сообщением об ошибке.
func HandleFatalError(err error, logger *zap.Logger) {
	if logger == nil {
		panic("logger is nil")
	}
	if err != nil {
		logger.Fatal("Ошибка", zap.Error(err))
	}
}
