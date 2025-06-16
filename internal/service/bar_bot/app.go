package bar_bot

import (
	"sync"
	"time"

	"tg_seller/internal/domain"

	"go.uber.org/zap"
)

type BarBot struct {
	logger       *zap.Logger
	SheetService domain.SheetService
	UserRepo     domain.UserRepo

	ticker        *time.Ticker
	forceUpdateCh chan struct{}
	stopCh        chan struct{}
	mu            sync.Mutex
}

func NewBarBot(sheetService domain.SheetService, userRepo domain.UserRepo, logger *zap.Logger, forceUpdateCh chan struct{}) *BarBot {
	bot := &BarBot{
		logger:        logger,
		SheetService:  sheetService,
		UserRepo:      userRepo,
		ticker:        time.NewTicker(10 * time.Minute),
		forceUpdateCh: forceUpdateCh,
		stopCh:        make(chan struct{}),
	}
	go bot.backgroundSync()
	return bot
}

// Фоновая синхронизация неотправленных клиентов
func (b *BarBot) backgroundSync() {
	// Сразу синхронизируем при старте
	b.syncUnsyncedClients()
	for {
		select {
		case <-b.ticker.C:
			b.syncUnsyncedClients()
		case <-b.forceUpdateCh:
			b.syncUnsyncedClients()
		case <-b.stopCh:
			b.ticker.Stop()
			return
		}
	}
}

// Синхронизировать неотправленных клиентов
func (b *BarBot) syncUnsyncedClients() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logger.Info("начинаем синхронизацию неотправленных клиентов")

	clients, err := b.UserRepo.GetUnsyncedClients()
	if err != nil {
		b.logger.Error("ошибка получения несинхронизированных клиентов", zap.Error(err))
		return
	}
	b.logger.Info("получены несинхронизированные клиенты", zap.Int("количество", len(clients)))

	if len(clients) == 0 {
		b.logger.Info("нет клиентов для синхронизации")
		return
	}

	for _, client := range clients {
		b.logger.Info("обработка клиента",
			zap.String("имя", client.Name),
			zap.String("телефон", client.Phone),
			zap.String("бар", client.Bar))

		row, err := b.SheetService.FindFirstFreeRow()
		if err != nil {
			b.logger.Error("ошибка поиска свободной строки для клиента",
				zap.Error(err),
				zap.String("телефон", client.Phone),
				zap.String("имя", client.Name))
			continue
		}
		b.logger.Info("найдена свободная строка", zap.Int("номер_строки", row))

		if row <= 1 {
			row = 2 // строка 1 — заголовки, данные с 2-й
			b.logger.Info("корректировка номера строки на 2 (первая строка для заголовков)")
		}

		b.logger.Info("попытка вставки клиента в таблицу",
			zap.Int("строка", row),
			zap.String("имя", client.Name))

		err = b.SheetService.InsertClient(row, client)
		if err != nil {
			b.logger.Error("ошибка вставки клиента в таблицу",
				zap.Error(err),
				zap.String("телефон", client.Phone),
				zap.Int("строка", row))
			continue
		}
		b.logger.Info("клиент успешно добавлен в таблицу",
			zap.String("имя", client.Name),
			zap.Int("строка", row))

		err = b.UserRepo.UpdateSheetIsSynced(client.ID, true)
		if err != nil {
			b.logger.Error("ошибка обновления статуса синхронизации",
				zap.Error(err),
				zap.String("телефон", client.Phone),
				zap.Uint("id", client.ID))
		} else {
			b.logger.Info("статус синхронизации успешно обновлен",
				zap.String("имя", client.Name),
				zap.Uint("id", client.ID))
		}
	}

	b.logger.Info("синхронизация завершена")
}

// ForceUpdate немедленно запускает синхронизацию
func (b *BarBot) ForceUpdate() {
	select {
	case b.forceUpdateCh <- struct{}{}:
	default:
	}
}

// Остановка фоновой задачи (по желанию)
func (b *BarBot) Stop() {
	close(b.stopCh)
}
