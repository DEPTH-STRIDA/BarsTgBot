package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// ConfigFile структура предоставляет путь к файлу, а также указатель на структуру.
// Config - указатель на структуру.
// Структуры могу содержать теги envconfig и godotenv.
type ConfigFile struct {
	// Путь к файлу.
	Path string
	// Конфигурация - указатель на структуру.
	Config interface{}
}

// LoadConfigFiles предоставляет возможность загрузки сразу нескольких конфигурационных файлов
// и анмаршалинга в структуры.
// Структуры могу содержать теги envconfig и godotenv.
func LoadConfigFiles(configFiles ...*ConfigFile) error {
	for _, configFile := range configFiles {
		if configFile.Path != "" {
			if err := godotenv.Load(configFile.Path); err != nil {
				return err
			}
		}

		err := envconfig.Process("", configFile.Config)
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadConfigs предоставляет возможность загрузки сразу нескольких конфигураций и анмаршалинга в структуры.
//   - config - ссылки на структуры. Структуры могу содержать теги envconfig и godotenv.
func LoadConfigs(config ...interface{}) error {
	for _, cfg := range config {
		err := envconfig.Process("", cfg)
		if err != nil {
			return err
		}
	}
	return nil
}
