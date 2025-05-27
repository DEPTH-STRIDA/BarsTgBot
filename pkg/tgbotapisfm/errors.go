package tgbotapisfm

import "fmt"

var (
	// ErrStatesNil возникает когда карта состояний nil
	ErrStatesNil = fmt.Errorf("states map is nil")

	// ErrInvalidToken возникает при пустом или невалидном токене
	ErrInvalidToken = fmt.Errorf("invalid bot token")

	// ErrNegativeExpiration возникает при отрицательном времени хранения
	ErrNegativeExpiration = fmt.Errorf("expiration time cannot be negative")

	// ErrNegativeCleanup возникает при отрицательном интервале очистки
	ErrNegativeCleanup = fmt.Errorf("cleanup interval cannot be negative")

	// ErrTelegramInit возникает при ошибке инициализации Telegram API
	ErrTelegramInit = fmt.Errorf("failed to initialize telegram bot api")

	// ErrBotStarted возникает при попытке изменить конфигурацию запущенного бота
	ErrBotStarted = fmt.Errorf("cannot modify running bot")

	// ErrStateNotFound возникает когда состояние не найдено в кеше
	ErrStateNotFound = fmt.Errorf("user state not found")

	// ErrInvalidStateType возникает при ошибке приведения типа состояния
	ErrInvalidStateType = fmt.Errorf("invalid state type in cache")

	// ErrStateHandlerNotFound возникает когда обработчик для состояния не найден
	ErrStateHandlerNotFound = fmt.Errorf("state handler not found")

	// ErrSendMessageFailed возникает при неудачных попытках отправки сообщения
	ErrSendMessageFailed = fmt.Errorf("all attempts to send message failed")

	// ErrEditMessageFailed возникает при неудачных попытках редактирования сообщения
	ErrEditMessageFailed = fmt.Errorf("all attempts to edit message failed")

	// ErrDeleteMessageFailed возникает при неудачных попытках удалить сообщения
	ErrDeleteMessageFailed = fmt.Errorf("all attempts to delete message failed")
)

// ValidationError представляет ошибку валидации с дополнительной информацией
type ValidationError struct {
	Err   error
	Value interface{}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%v: %v", e.Err, e.Value)
}

// NewValidationError создает новую ошибку валидации
func NewValidationError(err error, value interface{}) error {
	return &ValidationError{
		Err:   err,
		Value: value,
	}
}
