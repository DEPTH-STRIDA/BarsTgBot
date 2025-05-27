package tgbotapisfm

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerFunc func(b *Bot, u tgbotapi.Update) error

type Handler struct {
	// Handle обрабатывает входящее обновление от Telegram.
	Handle HandlerFunc

	// Description описание обработчика.
	Description string
}

// State представляет состояние бота и определяет правила обработки сообщений.
type State struct {
	// Если true, триггеры обработчиков проверяются независимо от текущего состояния пользователя
	// Если в глобальном состоянии найден подходящий обработчик, то он выполняется, а тригеры другого
	// состояния не выполняются.
	// После первого подходящего глобального состояния, другие глобальные состояния не выполняются.
	// Глобальные состояния устанавливаются один раз при иницилизации.
	Global bool
	// Выполняется при входе в состояние. Каждый раз, когда от пользователя приходит обновление, а пользователь находится в состоянии.
	AtEntranceFunc *Handler
	// Выполняется для всех событий, которые не попали в маршруты.
	CatchAllFunc *Handler
	// Сопоставляет текст сообщения с ключом обработчика и выполняет его.
	// Текст пользователя переводится в lowercase, поэтому ключи должны быть в таком же формате.
	MessageHandlers map[string]Handler
	// Сопоставляет текст сообщения с ключом обработчика и выполняет его
	CallbackHandlers map[string]Handler
}

// NewState создает новый экземпляр State с заданными параметрами.
func NewState(global bool, atEntranceFunc *Handler, catchAllFunc *Handler) *State {
	return &State{
		Global:           global,
		AtEntranceFunc:   atEntranceFunc,
		CatchAllFunc:     catchAllFunc,
		MessageHandlers:  make(map[string]Handler),
		CallbackHandlers: make(map[string]Handler),
	}
}
