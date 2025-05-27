package tg

import (
	"fmt"
	"strings"
	"tg_seller/internal/domain"
	"tg_seller/internal/model"
	"tg_seller/pkg/tgbotapisfm"
	"time"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gocache "github.com/patrickmn/go-cache"
)

type TGHandler struct {
	UserRepo    domain.UserRepo
	cache       *gocache.Cache
	bot         *tgbotapisfm.Bot
	forceUpdate chan struct{}
}

type Cache struct {
	UserId int64
	Bar    string
	Name   string
	Phone  string
}

func NewTGHandler(bot *tgbotapisfm.Bot, forceUpdate chan struct{}, userRepo domain.UserRepo) *TGHandler {
	cache := gocache.New(24*time.Hour, 1*time.Hour)
	return &TGHandler{
		cache:       cache,
		bot:         bot,
		forceUpdate: forceUpdate,
		UserRepo:    userRepo,
	}
}

func (h *TGHandler) SetBot(bot *tgbotapisfm.Bot) {
	h.bot = bot
}

func (h *TGHandler) StartState() tgbotapisfm.State {
	var StartState = tgbotapisfm.State{
		Global: true,
		MessageHandlers: map[string]tgbotapisfm.Handler{
			"/start":        h.StartHandler(),
			"black cat pub": h.BarSelectHandler("Black cat pub"),
			"bar heroes":    h.BarSelectHandler("Bar Heroes"),
		},
	}
	return StartState
}

func (h *TGHandler) StartHandler() tgbotapisfm.Handler {
	var StartHandler = tgbotapisfm.Handler{
		Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите бар")
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				[]tgbotapi.KeyboardButton{
					tgbotapi.NewKeyboardButton("Black cat pub"),
					tgbotapi.NewKeyboardButton("Bar Heroes"),
				},
			)
			_, err := bot.SendMessage(msg)
			return err
		},
	}
	return StartHandler
}

func (h *TGHandler) BarSelectHandler(bar string) tgbotapisfm.Handler {
	return tgbotapisfm.Handler{
		Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
			h.SaveBarToCache(update.Message.From.ID, bar)
			bot.SetUserState(update.Message.From.ID, "name_enter")
			return h.NameEnterNameState().AtEntranceFunc.Handle(bot, update)
		},
	}
}

func (h *TGHandler) SaveBarToCache(userId int64, bar string) {
	var cacheData Cache
	if x, found := h.cache.Get(fmt.Sprint(userId)); found {
		cacheData, _ = x.(Cache)
	}
	cacheData.UserId = userId
	cacheData.Bar = bar
	h.cache.Set(fmt.Sprint(userId), cacheData, gocache.DefaultExpiration)
}

func (h *TGHandler) SaveNameToCache(userId int64, name string) {
	// Получаем из кеша, если есть
	var cacheData Cache
	if x, found := h.cache.Get(fmt.Sprint(userId)); found {
		cacheData, _ = x.(Cache)
	}
	cacheData.UserId = userId
	cacheData.Name = name
	h.cache.Set(fmt.Sprint(userId), cacheData, gocache.DefaultExpiration)
}

func normalizeName(name string) string {
	parts := strings.Fields(name)
	for i, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			for j := 1; j < len(runes); j++ {
				runes[j] = unicode.ToLower(runes[j])
			}
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, " ")
}

func (h *TGHandler) NameEnterNameState() tgbotapisfm.State {
	var NameEnterState = tgbotapisfm.State{
		Global: false,

		AtEntranceFunc: &tgbotapisfm.Handler{
			Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите ваши имя и фамилию")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				_, err := bot.SendMessage(msg)
				return err
			},
		},
		CatchAllFunc: &tgbotapisfm.Handler{
			Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
				name := strings.TrimSpace(update.Message.Text)
				if len(name) > 255 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Имя слишком длинное, введите не более 255 символов")
					_, _ = bot.SendMessage(msg)
					return nil
				}
				parts := strings.Fields(name)
				if len(parts) < 2 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, введите имя и фамилию через пробел")
					_, _ = bot.SendMessage(msg)
					return nil
				}
				// Проверка только на буквы
				valid := true
				for _, part := range parts {
					if len(part) < 2 || !isCyrillicOrLatin(part) {
						valid = false
						break
					}
				}
				if !valid {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Имя и фамилия должны состоять только из букв")
					_, _ = bot.SendMessage(msg)
					return nil
				}
				normalized := normalizeName(name)
				h.SaveNameToCache(update.Message.From.ID, normalized)

				text := fmt.Sprintf("*Ваше имя:* _%s_\n\n"+
					"Если все верно, нажмите *Продолжить*\\."+
					"\nЕсли хотите изменить имя, просто отправьте новое\\.",
					escapeMarkdown(normalized))

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ParseMode = "MarkdownV2"
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					[]tgbotapi.KeyboardButton{
						tgbotapi.NewKeyboardButton("Продолжить"),
					},
				)
				_, _ = bot.SendMessage(msg)
				return nil
			},
		},
		MessageHandlers: map[string]tgbotapisfm.Handler{
			"продолжить": h.NameEnterPhoneContinueHandler(),
		},
	}
	return NameEnterState
}

func isCyrillicOrLatin(s string) bool {
	for _, r := range s {
		if !(unicode.Is(unicode.Cyrillic, r) || unicode.IsLetter(r)) {
			return false
		}
	}
	return true
}

func (h *TGHandler) NameEnterPhoneState() tgbotapisfm.State {
	var PhoneEnterState = tgbotapisfm.State{
		Global: false,
		AtEntranceFunc: &tgbotapisfm.Handler{
			Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
				text := "*Введите номер телефона*\n" +
					"_Только для номеров РФ_"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ParseMode = "MarkdownV2"
				_, err := bot.SendMessage(msg)
				return err
			},
		},
		CatchAllFunc: &tgbotapisfm.Handler{
			Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
				phoneRaw := update.Message.Text
				phone := extractDigits(phoneRaw)
				if len(phone) < 10 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Некорректный номер. Введите номер телефона РФ (минимум 10 цифр)")
					_, _ = bot.SendMessage(msg)
					return nil
				}
				// Оставляем только последние 10 цифр
				if len(phone) > 10 {
					phone = phone[len(phone)-10:]
				}
				formatted := formatPhone(phone)
				// Сохраняем в кеш
				var cacheData Cache
				if x, found := h.cache.Get(fmt.Sprint(update.Message.From.ID)); found {
					cacheData, _ = x.(Cache)
				}
				cacheData.UserId = update.Message.From.ID
				cacheData.Phone = phone
				h.cache.Set(fmt.Sprint(update.Message.From.ID), cacheData, gocache.DefaultExpiration)

				text := fmt.Sprintf("*Ваш номер:* _%s_\n\n"+
					"Если все верно, нажмите *Завершить регистрацию*\\."+
					"\nЕсли хотите изменить номер, просто отправьте новый\\.",
					escapeMarkdown(formatted))

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ParseMode = "MarkdownV2"
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
					[]tgbotapi.KeyboardButton{
						tgbotapi.NewKeyboardButton("Завершить регистрацию"),
					},
				)
				_, _ = bot.SendMessage(msg)
				return nil
			},
		},
		MessageHandlers: map[string]tgbotapisfm.Handler{
			"завершить регистрацию": h.RegistrationFinishHandler(),
		},
	}
	return PhoneEnterState
}

func (h *TGHandler) RegistrationFinishHandler() tgbotapisfm.Handler {
	return tgbotapisfm.Handler{
		Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
			var cacheData Cache
			if x, found := h.cache.Get(fmt.Sprint(update.Message.From.ID)); found {
				cacheData, _ = x.(Cache)
			}

			// Проверяем, что имя и телефон есть и валидны
			if cacheData.Name == "" || len(strings.Fields(cacheData.Name)) < 2 || len(cacheData.Name) > 255 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Имя некорректно. Введите имя и фамилию заново.")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				_, _ = bot.SendMessage(msg)
				return nil
			}
			if cacheData.Phone == "" || len(cacheData.Phone) != 10 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Телефон некорректен. Введите номер заново.")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				_, _ = bot.SendMessage(msg)
				return nil
			}

			// Проверяем, существует ли уже клиент с таким телефоном в этом баре
			exists, err := h.UserRepo.ExistsByPhoneAndBar(cacheData.Phone, cacheData.Bar)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Произошла ошибка при проверке данных. Попробуйте позже.")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				_, _ = bot.SendMessage(msg)
				return nil
			}
			if exists {
				text := fmt.Sprintf("❗ Вы уже зарегистрированы в баре *%s* "+
					"с номером _%s_\\.",
					escapeMarkdown(cacheData.Bar),
					escapeMarkdown(formatPhone(cacheData.Phone)))
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ParseMode = "MarkdownV2"
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				_, _ = bot.SendMessage(msg)
				return nil
			}

			// Добавляем в БД
			client := &model.Client{
				Name:           cacheData.Name,
				Phone:          cacheData.Phone,
				Bar:            cacheData.Bar,
				Username:       update.Message.From.UserName,
				RegistrationAt: time.Now().Format("02.01.2006 15:04"),
			}
			err = h.UserRepo.InsertClient(client)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при сохранении в базу. Попробуйте позже.")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				_, _ = bot.SendMessage(msg)
				return nil
			}

			// Отправляем сигнал в канал
			select {
			case h.forceUpdate <- struct{}{}:
			default:
			}

			text := "✅ *Регистрация успешно завершена\\!*\n\n" +
				fmt.Sprintf("📍 *Бар:* %s\n"+
					"👤 *Имя:* _%s_\n"+
					"📱 *Телефон:* _%s_\n\n"+
					"Спасибо за регистрацию\\!",
					escapeMarkdown(cacheData.Bar),
					escapeMarkdown(cacheData.Name),
					escapeMarkdown(formatPhone(cacheData.Phone)))

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			msg.ParseMode = "MarkdownV2"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			_, _ = bot.SendMessage(msg)
			return nil
		},
	}
}

func extractDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatPhone(phone string) string {
	if len(phone) != 10 {
		return phone
	}
	return fmt.Sprintf("+7 (%s) %s-%s-%s", phone[:3], phone[3:6], phone[6:8], phone[8:10])
}

func (h *TGHandler) NameEnterPhoneContinueHandler() tgbotapisfm.Handler {
	return tgbotapisfm.Handler{
		Handle: func(bot *tgbotapisfm.Bot, update tgbotapi.Update) error {
			// Повторно проверяем имя из кеша
			var cacheData Cache
			if x, found := h.cache.Get(fmt.Sprint(update.Message.From.ID)); found {
				cacheData, _ = x.(Cache)
			}
			if cacheData.Name == "" || len(strings.Fields(cacheData.Name)) < 2 || len(cacheData.Name) > 255 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Имя некорректно. Введите имя и фамилию заново.")
				_, _ = bot.SendMessage(msg)
				return nil
			}
			// Всё ок — переводим в состояние phone_enter
			bot.SetUserState(update.Message.From.ID, "phone_enter")
			return h.NameEnterPhoneState().AtEntranceFunc.Handle(bot, update)
		},
	}
}

func (h *TGHandler) StatesMap() map[string]tgbotapisfm.State {
	return map[string]tgbotapisfm.State{
		"start":       h.StartState(),
		"name_enter":  h.NameEnterNameState(),
		"phone_enter": h.NameEnterPhoneState(),
	}
}

// Добавляем функцию для экранирования специальных символов Markdown
func escapeMarkdown(text string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	escaped := text
	for _, char := range specialChars {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}
	return escaped
}
