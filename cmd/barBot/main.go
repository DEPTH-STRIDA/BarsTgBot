package main

import (
	"tg_seller/internal/config"
	"tg_seller/internal/model"
	user_ps "tg_seller/internal/repository/postgres"
	"tg_seller/internal/service/bar_bot"
	"tg_seller/internal/service/sheet"
	"tg_seller/internal/service/tg"
	pkg_config "tg_seller/pkg/config"
	"tg_seller/pkg/db/postgres"
	"tg_seller/pkg/masker"
	"tg_seller/pkg/tgbotapisfm"
	"tg_seller/pkg/zaplogger"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
func main() {
	logger, err := zaplogger.New()
	if err != nil {
		panic(err)
	}

	cfg := config.Config{}
	if err := pkg_config.LoadConfigs(&cfg); err != nil {
		logger.Fatal("error loading configs", zap.Error(err))
	}

	if err := masker.LogConfigs(logger, &cfg); err != nil {
		logger.Fatal("error logging configs", zap.Error(err))
	}

	dbGorm, err := postgres.NewGormConnection(cfg.DBConfig)
	if err != nil {
		logger.Fatal("error creating gorm connection", zap.Error(err))
	}
	dbGorm.AutoMigrate(&model.Client{})
	userRepo := user_ps.NewClientRepository(dbGorm)

	sheetService, err := sheet.NewSheetService(
		cfg.GoogleSheetConfig.CredentialsBase64,
		cfg.GoogleSheetConfig.SheetID,
		cfg.GoogleSheetConfig.ClientListID,
		cfg.GoogleSheetConfig.PauseMs,
		sheet.NewDefaultColumnMap(),
	)
	if err != nil {
		logger.Fatal("error creating sheet service", zap.Error(err))
	}

	forceUpdate := make(chan struct{}, 1)

	tgHandler := tg.NewTGHandler(nil, forceUpdate, userRepo)
	mapStates := tgHandler.StatesMap()

	bot, err := tgbotapisfm.NewBot(tgbotapisfm.Config{
		Token:           cfg.TelegramConfig.BotToken,
		Expiration:      24 * time.Hour,
		CleanupInterval: 1 * time.Hour,
		States:          mapStates,
	}, []int64{}, logger)
	if err != nil {
		logger.Fatal("error creating bot", zap.Error(err))
	}

	tgHandler.SetBot(bot)

	_ = bar_bot.NewBarBot(sheetService, userRepo, logger, forceUpdate)

	// Запускаем бота в основной горутине
	errChan := bot.Start(30, 0)
	if err := <-errChan; err != nil {
		logger.Fatal("error starting bot", zap.Error(err))
	}
}
