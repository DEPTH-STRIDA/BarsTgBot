package config_loader

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

// LoadEnv загружает конфигурацию из файла .env в любую структуру
func LoadEnv(path string, cfg interface{}, logger *zap.Logger) error {
	if err := godotenv.Load(path); err != nil {
		logger.Info("No .env file found or failed to load, using environment variables: ", zap.Error(err))
	}

	if err := envconfig.Process("", cfg); err != nil {
		return err
	}

	return nil
}
