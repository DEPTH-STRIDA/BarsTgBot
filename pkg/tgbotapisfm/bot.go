package tgbotapisfm

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gocache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapLogger создает новый асинхронный JSON-логгер с настроенным форматированием временных меток,
// уровней логирования и информации о вызовах.
func NewZapLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Encoding = "json"
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	return config.Build(zap.AddCallerSkip(1))
}

// Config структура для конфигурации бота
type Config struct {
	Token           string           // Токен бота
	Expiration      time.Duration    // Время хранения состояний пользователя
	CleanupInterval time.Duration    // Интервал очистки кеша
	States          map[string]State // Карта состояний
}

// Bot структура для бота
type Bot struct {
	BotAPI        *tgbotapi.BotAPI // API бота. Экспортируется для доступа к нему из вне
	expiration    time.Duration    // Время хранения состояний пользователя
	limiter       *Limiter         // Лимитер для ограничения количества запросов к API
	cache         *gocache.Cache   // Кеш для хранения состояний пользователей
	logger        *zap.Logger      // Логгер для записи событий
	states        map[string]State // Состояния пользователя
	globalStates  []*State         // Состояния, в которые может перейти пользователь из любого другоо
	updateHandler HandlerFunc      // Обработчик, который будет вызываться при получении любого обновления
	mu            sync.RWMutex     // Мьютекс для проверки состояния бота
	statesMu      sync.RWMutex     // Мьютекс для безопасного обновления состояний

	IgnoreList []int64 // Список ID пользователей, которые будут игнорироваться
}

// NewBot конструктор нового бота
// logger - необяхательный параметр, если не передан, то будет создан новый логгер
func NewBot(config Config, ignoreList []int64, logger ...*zap.Logger) (*Bot, error) {
	// Если карта состояний пуста, то нужно ее иницилизировать, чтобы избежать ошибок
	if config.States == nil {
		config.States = make(map[string]State)
	}
	if config.Expiration < 0 {
		return nil, NewValidationError(ErrNegativeExpiration, config.Expiration)
	}
	if config.CleanupInterval < 0 {
		return nil, NewValidationError(ErrNegativeCleanup, config.CleanupInterval)
	}
	if config.Token == "" {
		return nil, ErrInvalidToken
	}

	botAPI, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, NewValidationError(ErrTelegramInit, err)
	}
	// Дополнительная map'a глобальных состояний
	globalStates := make([]*State, 0)
	for _, state := range config.States {
		if state.Global {
			globalStates = append(globalStates, &state)
		}
	}

	var zapLogger *zap.Logger
	if len(logger) > 0 {
		zapLogger = logger[0]
	} else {
		zapLogger, err = NewZapLogger()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}
	app := Bot{
		BotAPI:       botAPI,
		limiter:      NewLimiter(),
		cache:        gocache.New(config.Expiration, config.CleanupInterval),
		states:       config.States,
		globalStates: globalStates,
		expiration:   config.Expiration,
		logger:       zapLogger,
		IgnoreList:   ignoreList,
	}

	return &app, nil
}

// SetLogger заменяет текущий логгер
// Должен вызываться до Start()
func (b *Bot) SetLogger(logger *zap.Logger) error {
	if !b.mu.TryRLock() {
		return NewValidationError(ErrBotStarted, "logger")
	}
	defer b.mu.RUnlock()

	b.logger = logger
	return nil
}

// SetUpdateHandler устанавливает обработчик обновлений
// Должен вызываться до Start()
func (b *Bot) SetUpdateHandler(handler HandlerFunc) error {
	if !b.mu.TryRLock() {
		return NewValidationError(ErrBotStarted, "update handler")
	}
	defer b.mu.RUnlock()

	b.updateHandler = handler
	return nil
}

// Start запускает обработку обновлений в горутине и возвращает канал для ошибок
func (b *Bot) Start(offset, timeout int) chan error {
	errChan := make(chan error, 1)

	if !b.mu.TryLock() {
		b.logger.Warn("Бот уже запущен")
		errChan <- fmt.Errorf("bot is already running")
		return errChan
	}

	b.logger.Info("Запуск бота")
	go func() {
		if err := b.HandleUpdates(offset, timeout); err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	return errChan
}

// Stop останавливает обработку обновлений
func (b *Bot) Stop() {
	b.mu.Unlock()                   // Разблокируем мьютекс, заблокированный в Start()
	b.BotAPI.StopReceivingUpdates() // Останавливаем получение обновлений
	b.logger.Info("Остановка обработки обновлений")
}

// HandleUpdates запускает обработку всех обновлений поступающих боту из телеграмма
func (app *Bot) HandleUpdates(offset, timeout int) error {
	// Настройка обновлений
	u := tgbotapi.NewUpdate(offset)
	u.Timeout = timeout
	updates := app.BotAPI.GetUpdatesChan(u)
	app.logger.Info("Запуск обработки обновлений")

	for update := range updates {
		// Обработка любого обновления
		if app.updateHandler != nil {
			if err := app.updateHandler(app, update); err != nil {
				app.logger.Error("Ошибка в обработчике обновлений", zap.Error(err))
				return fmt.Errorf("update handler error: %w", err)
			}
		}

		// Обработка локальных стейтов
		if update.SentFrom() == nil {
			continue
		}
		if slices.Contains(app.IgnoreList, update.SentFrom().ID) {
			continue
		}
		if update.FromChat() != nil && slices.Contains(app.IgnoreList, update.FromChat().ID) {
			continue
		}

		// Обработка глобальных стейтов
		globalStateFound, err := app.HandleGlobalStates(update)
		if err != nil {
			app.logger.Error("failed to handle global state", zap.Error(err))
			return fmt.Errorf("global state error: %w", err)
		}
		// Если глобальное состояние найдено, то продолжаем
		if globalStateFound {
			continue
		}

		// Получение названия состояния пользователя
		userStateName, err := app.GetUserState(update.SentFrom().ID)
		if err != nil {
			app.logger.Error("failed to get user state", zap.Error(err))
			continue
		}

		// Получение состояния
		userState, ok := app.states[userStateName]
		if !ok {
			app.logger.Error("state not found in states map", zap.String("state", userStateName))
			return fmt.Errorf("state %s not found", userStateName)
		}

		// Обработка обновления по локальному состоянию
		_, err = app.SelectHandler(update, &userState)
		if err != nil {
			app.logger.Error("failed to handle user state", zap.Error(err))
			return fmt.Errorf("handle user state error: %w", err)
		}
	}

	return nil
}

// GetUserState возвращает название состояния, в котором находится пользователь
func (app *Bot) GetUserState(userId int64) (string, error) {
	userStateInterface, ok := app.cache.Get(strconv.FormatInt(userId, 10))
	if !ok {
		return "", ErrStateNotFound
	}

	userState, ok := userStateInterface.(string)
	if !ok {
		return "", ErrInvalidStateType
	}

	return userState, nil
}

// SetUserState меняет состояние пользователя
func (app *Bot) SetUserState(userId int64, state string) error {
	app.statesMu.RLock()
	_, ok := app.states[state]
	app.statesMu.RUnlock()

	if !ok {
		return NewValidationError(ErrStateHandlerNotFound, state)
	}

	app.cache.Set(strconv.FormatInt(userId, 10), state, app.expiration)
	return nil
}

// SetUserStateImmediate меняет состояние пользователя и сразу обрабатывает текущее обновление
func (app *Bot) SetUserStateImmediate(userId int64, state string, update tgbotapi.Update) error {
	if err := app.SetUserState(userId, state); err != nil {
		return err
	}

	app.statesMu.RLock()
	newState, ok := app.states[state]
	app.statesMu.RUnlock()

	if ok {
		// Вызываем действие при входе, если оно есть и это не глобальное состояние
		if newState.AtEntranceFunc != nil {
			if err := newState.AtEntranceFunc.Handle(app, update); err != nil {
				app.logger.Error("failed to handle entrance function", zap.Error(err))
			}
		}

		// Немедленная обработка текущего обновления
		_, err := app.SelectHandler(update, &newState)
		if err != nil {
			app.logger.Error("failed to handle immediate reaction", zap.Error(err))
		}
	}

	return nil
}

// HandleGlobalStates проверяет подходит ли действие пользователя под
// глобальные состояния и если подходит, то выполняет его.
// Возвращает true, если обработчик нашелся и выполнился.
func (app *Bot) HandleGlobalStates(update tgbotapi.Update) (bool, error) {
	app.statesMu.RLock()
	defer app.statesMu.RUnlock()

	// Обработка всех глобальных состояний
	for _, state := range app.globalStates {
		// Обработка состояния
		handlerIsFound, err := app.SelectHandler(update, state)
		// Если ошибка, то пропускаем состояние
		if err != nil {
			app.logger.Error("failed to handle global state", zap.Error(err))
			continue
		}
		// Если обработчик найден, то возвращаем true
		if handlerIsFound {
			return true, nil
		}
	}
	return false, nil
}

func (app *Bot) SelectHandler(update tgbotapi.Update, userState *State) (bool, error) {
	switch {
	case update.Message != nil:
		if userState.MessageHandlers != nil {
			return app.handleMessage(userState, update)
		} else {
			app.logger.Info("command not found",
				zap.String("command", update.Message.Text),
				zap.Int64("chat_id", update.Message.Chat.ID),
				zap.String("username", update.Message.Chat.UserName),
			)
			return false, nil
		}
	case update.CallbackQuery != nil:
		if userState.CallbackHandlers != nil {
			app.logger.Info("callback found",
				zap.String("callback", update.CallbackQuery.Data),
				zap.Int64("user_id", update.CallbackQuery.From.ID),
				zap.String("username", update.CallbackQuery.From.UserName),
			)
			return app.handleCallback(userState, update)
		} else {
			app.logger.Info("callback not found",
				zap.String("callback", update.CallbackQuery.Data),
				zap.Int64("user_id", update.CallbackQuery.From.ID),
				zap.String("username", update.CallbackQuery.From.UserName),
			)
			return false, nil
		}
	}
	return false, nil
}

// handleMessage ищет команду в map'е и выполняет ее
func (app *Bot) handleMessage(userState *State, update tgbotapi.Update) (bool, error) {
	messageFound := false

	// Поиск обработчика
	if currentAction, ok := userState.MessageHandlers[strings.ToLower(strings.TrimSpace(update.Message.Text))]; ok {
		messageFound = true
		if err := currentAction.Handle(app, update); err != nil {
			app.logger.Error("failed to handle command", zap.Error(err))
		} else {
			app.logger.Info("command handled successfully",
				zap.String("command", update.Message.Text),
				zap.Int64("chat_id", update.Message.Chat.ID),
				zap.String("username", update.Message.Chat.UserName),
			)
		}
	} else {
		if userState.CatchAllFunc != nil {
			err := userState.CatchAllFunc.Handle(app, update)
			if err != nil {
				app.logger.Error("failed to handle command", zap.Error(err))
			}
		} else {
			app.logger.Info("command not found",
				zap.String("command", update.Message.Text),
				zap.Int64("chat_id", update.Message.Chat.ID),
				zap.String("username", update.Message.Chat.UserName),
			)
		}

	}
	return messageFound, nil
}

// handleCallback ищет команду в map'е и выполняет ее
func (app *Bot) handleCallback(userState *State, update tgbotapi.Update) (bool, error) {
	callbackFound := false

	if currentAction, ok := userState.CallbackHandlers[update.CallbackQuery.Data]; ok {
		callbackFound = true
		if err := currentAction.Handle(app, update); err != nil {
			app.logger.Error("failed to handle callback", zap.Error(err))
			return callbackFound, err
		}

		app.logger.Info("callback handled successfully",
			zap.String("callback", update.CallbackQuery.Data),
			zap.Int64("user_id", update.CallbackQuery.From.ID),
			zap.String("username", update.CallbackQuery.From.UserName),
		)
	} else {
		if userState.CatchAllFunc != nil {
			err := userState.CatchAllFunc.Handle(app, update)
			if err != nil {
				app.logger.Error("failed to handle callback", zap.Error(err))
			}
		} else {
			app.logger.Info("callback not found",
				zap.Int64("user_id", update.CallbackQuery.From.ID),
				zap.String("username", update.CallbackQuery.From.UserName),
				zap.String("callback", update.CallbackQuery.Data),
			)
		}
	}
	return callbackFound, nil
}

// ReplaceStates безопасно заменяет все состояния бота на новые
func (b *Bot) ReplaceStates(newStates map[string]State) {
	b.statesMu.Lock()
	defer b.statesMu.Unlock()

	// Создаем новый слайс для глобальных состояний
	newGlobalStates := make([]*State, 0)
	for _, state := range newStates {
		if state.Global {
			stateCopy := state // Создаем копию, чтобы не хранить указатель на переменную цикла
			newGlobalStates = append(newGlobalStates, &stateCopy)
		}
	}

	b.states = newStates
	b.globalStates = newGlobalStates
	b.logger.Info("Состояния бота успешно обновлены")
}

// EditMessageMedia редактирует медиа в сообщении
func (b *Bot) EditMessageMedia(config tgbotapi.EditMessageMediaConfig) (tgbotapi.Message, error) {
	return b.BotAPI.Send(config)
}
