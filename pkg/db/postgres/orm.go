package postgres

import (
	"fmt"
	"tg_seller/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewGormConnection создает новое соединение с базой данных через GORM
func NewGormConnection(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"user=%s password=%s host=%s dbname=%s port=%s sslmode=%s",
		cfg.User, cfg.Pass, cfg.Host, cfg.DBName, cfg.Port, cfg.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
