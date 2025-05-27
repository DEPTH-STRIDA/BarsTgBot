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
	clients, err := b.UserRepo.GetUnsyncedClients()
	if err != nil {
		b.logger.Error("error getting unsynced clients", zap.Error(err))
		return
	}
	for _, client := range clients {
		row, err := b.SheetService.FindFirstFreeRow()
		if err != nil {
			b.logger.Error("error finding free row for client", zap.Error(err), zap.String("phone", client.Phone))
			continue
		}
		if row <= 1 {
			row = 2 // строка 1 — заголовки, данные с 2-й
		}
		err = b.SheetService.InsertClient(row, client)
		if err != nil {
			b.logger.Error("error inserting client to sheet", zap.Error(err), zap.String("phone", client.Phone))
			continue
		}
		err = b.UserRepo.UpdateSheetIsSynced(client.ID, true)
		if err != nil {
			b.logger.Error("error updating SheetIsSynced", zap.Error(err), zap.String("phone", client.Phone))
		}
	}
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
