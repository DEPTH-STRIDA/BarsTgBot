package config

import (
	"os"
	"testing"
)

type testConfig struct {
	Var1 string `envconfig:"VAR1"`
	Var2 string `envconfig:"VAR2"`
}

func TestLoadConfigFiles(t *testing.T) {
	// Создание временного файла с конфигурацией
	envContent := "VAR1=hello\nVAR2=world\n"
	tmpFile := "test.env"
	err := os.WriteFile(tmpFile, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("не удалось создать временный .env: %v", err)
	}
	defer os.Remove(tmpFile)

	var cfg testConfig
	configFile := &ConfigFile{
		Path:   tmpFile,
		Config: &cfg,
	}

	// Удаление переменных окружения
	os.Unsetenv("VAR1")
	os.Unsetenv("VAR2")

	// Загрузка конфигурации из файла, с помощью тестируемой функции
	err = LoadConfigFiles(configFile)
	if err != nil {
		t.Fatalf("LoadConfigFiles вернул ошибку: %v", err)
	}

	if cfg.Var1 != "hello" {
		t.Errorf("ожидали Var1=hello, получили %s", cfg.Var1)
	}
	if cfg.Var2 != "world" {
		t.Errorf("ожидали Var2=world, получили %s", cfg.Var2)
	}
}

func TestLoadConfigs(t *testing.T) {
	// Установка переменных окружения
	os.Setenv("VAR1", "foo")
	os.Setenv("VAR2", "bar")
	defer os.Unsetenv("VAR1")
	defer os.Unsetenv("VAR2")

	// Загрузка конфигурации из переменных окружения, с помощью тестируемой функции
	var cfg testConfig
	err := LoadConfigs(&cfg)
	if err != nil {
		t.Fatalf("LoadConfigs вернул ошибку: %v", err)
	}

	if cfg.Var1 != "foo" {
		t.Errorf("ожидали Var1=foo, получили %s", cfg.Var1)
	}
	if cfg.Var2 != "bar" {
		t.Errorf("ожидали Var2=bar, получили %s", cfg.Var2)
	}
}

func TestLoadConfigFiles_FileNotFound(t *testing.T) {
	var cfg testConfig
	configFile := &ConfigFile{
		Path:   "nonexistent.env",
		Config: &cfg,
	}
	err := LoadConfigFiles(configFile)
	if err == nil {
		t.Error("ожидали ошибку при отсутствии файла, но её не было")
	}
}
