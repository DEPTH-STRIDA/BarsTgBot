package tgbotapisfm

import (
	"sync"
	"time"
)

const (
	GlobalLimit    = 30          // Максимум запросов в секунду к API
	ChatLimit      = 1           // Максимум сообщений в секунду в один чат
	MultiChatLimit = 30          // Максимум сообщений в секунду в разные чаты
	WaitTime       = time.Second // Ждем полную секунду при превышении лимита
)

// Limiter управляет ограничениями запросов к API
type Limiter struct {
	mu sync.Mutex

	ChatTimes    map[int64]time.Time
	MessageTimes []time.Time
	ApiTimes     []time.Time
}

// NewLimiter создает новый лимитер
func NewLimiter() *Limiter {
	return &Limiter{
		ChatTimes:    make(map[int64]time.Time),
		MessageTimes: make([]time.Time, 0, MultiChatLimit),
		ApiTimes:     make([]time.Time, 0, GlobalLimit),
	}
}

// CheckMessage проверяет возможность отправки сообщения в чат
func (l *Limiter) CheckMessage(chatID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Проверяем ограничение на конкретный чат (1 сообщение в секунду)
	if lastTime, ok := l.ChatTimes[chatID]; ok {
		if wait := time.Second - now.Sub(lastTime); wait > 0 {
			time.Sleep(wait)
			now = time.Now()
		}
	}

	l.cleanup(now)

	// Проверяем общее ограничение на сообщения (30 в секунду)
	if len(l.MessageTimes) >= MultiChatLimit {
		oldest := l.MessageTimes[0]
		if wait := time.Second - now.Sub(oldest); wait > 0 {
			time.Sleep(wait)
			now = time.Now()
			l.cleanup(now)
		}
	}

	// Проверяем общее ограничение API (30 запросов в секунду)
	if len(l.ApiTimes) >= GlobalLimit {
		oldest := l.ApiTimes[0]
		if wait := time.Second - now.Sub(oldest); wait > 0 {
			time.Sleep(wait)
			now = time.Now()
			l.cleanup(now)
		}
	}

	// Отправка сообщения считается и как сообщение, и как запрос к API
	l.ChatTimes[chatID] = now
	l.MessageTimes = append(l.MessageTimes, now)
	l.ApiTimes = append(l.ApiTimes, now)
}

// CheckAPI проверяет возможность отправки запроса к API
func (l *Limiter) CheckAPI() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.cleanup(now)

	if len(l.ApiTimes) >= GlobalLimit {
		oldest := l.ApiTimes[0]
		if wait := time.Second - now.Sub(oldest); wait > 0 {
			time.Sleep(wait) // Ждем только до истечения самого старого запроса
			now = time.Now()
			l.cleanup(now)
		}
	}

	l.ApiTimes = append(l.ApiTimes, now)
}

// cleanup удаляет записи старше 1 секунды
func (l *Limiter) cleanup(now time.Time) {
	newMessageTimes := l.MessageTimes[:0]
	for _, t := range l.MessageTimes {
		if now.Sub(t) < time.Second {
			newMessageTimes = append(newMessageTimes, t)
		}
	}
	l.MessageTimes = newMessageTimes

	newAPITimes := l.ApiTimes[:0]
	for _, t := range l.ApiTimes {
		if now.Sub(t) < time.Second {
			newAPITimes = append(newAPITimes, t)
		}
	}
	l.ApiTimes = newAPITimes
}
