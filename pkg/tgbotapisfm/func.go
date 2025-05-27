package tgbotapisfm

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

func (b *Bot) SendDeleteMessage(msg tgbotapi.DeleteMessageConfig) (*tgbotapi.APIResponse, error) {
	b.limiter.CheckAPI()

	sendedMsg, err := b.BotAPI.Request(msg)
	if err != nil {
		return nil, err
	}

	return sendedMsg, nil
}

func (b *Bot) SendMessage(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	b.limiter.CheckMessage(msg.ChatID)

	sendedMsg, err := b.BotAPI.Send(msg)
	if err != nil {
		return sendedMsg, err
	}

	return sendedMsg, nil
}

// SendMessageRepet пытается отправить сообщение указанное количество раз
func (b *Bot) SendMessageRepet(msg tgbotapi.MessageConfig, numberRepetion int) (tgbotapi.Message, error) {
	for i := 0; i < numberRepetion; i++ {
		sendedMsg, err := b.SendMessage(msg)
		if err != nil {
			b.logger.Info("Ошибка при отправке сообщения с повтором",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", numberRepetion),
				zap.Error(err),
			)
			continue
		}
		return sendedMsg, nil
	}
	return tgbotapi.Message{}, NewValidationError(ErrSendMessageFailed, numberRepetion)
}

func (b *Bot) SendPinMessageEvent(messageID int, ChatID int64, disableNotification bool) (*tgbotapi.APIResponse, error) {
	b.limiter.CheckAPI()

	pinConfig := tgbotapi.PinChatMessageConfig{
		ChatID:              ChatID,
		MessageID:           messageID,
		DisableNotification: disableNotification,
	}

	APIResponse, err := b.BotAPI.Request(pinConfig)
	if err != nil {
		return nil, err
	}
	return APIResponse, nil
}

func (b *Bot) SendSticker(stickerID string, chatID int64) (*tgbotapi.Message, error) {
	b.limiter.CheckMessage(chatID)

	msg := tgbotapi.NewSticker(chatID, tgbotapi.FileID(stickerID))

	sendedMsg, err := b.BotAPI.Send(msg)
	if err != nil {
		return nil, err
	}

	return &sendedMsg, nil
}

func (b *Bot) SendUnPinAllMessageEvent(ChannelUsername string, chatID int64) (*tgbotapi.APIResponse, error) {
	b.limiter.CheckAPI()

	unpinConfig := tgbotapi.UnpinAllChatMessagesConfig{
		ChatID:          chatID,
		ChannelUsername: ChannelUsername,
	}

	APIresponse, err := b.BotAPI.Request(unpinConfig)
	if err != nil {
		return nil, err
	}

	return APIresponse, err
}

func (b *Bot) EditMessageRepet(editMsg tgbotapi.EditMessageTextConfig, numberRepetion int) (*tgbotapi.APIResponse, error) {
	var err error
	var response *tgbotapi.APIResponse

	for i := 0; i < numberRepetion; i++ {
		response, err = b.EditMessage(editMsg)
		if err != nil {
			b.logger.Info("Ошибка при редактировании сообщения с повтором",
				zap.Int("attempt", i),
				zap.Error(err),
			)
		} else {
			return response, nil
		}
	}
	return nil, NewValidationError(ErrEditMessageFailed, numberRepetion)
}

func (b *Bot) EditMessage(editMsg tgbotapi.EditMessageTextConfig) (*tgbotapi.APIResponse, error) {
	b.limiter.CheckAPI()

	response, err := b.BotAPI.Request(editMsg)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bot) DeleteMessageRepet(msgToDelete tgbotapi.DeleteMessageConfig, numberRepetion int) error {
	var err error

	for i := 0; i < numberRepetion; i++ {
		err = b.DeleteMessage(msgToDelete)
		if err != nil {
			b.logger.Info("Не удалось удалить сообщение из чата",
				zap.Int("attempt", i),
				zap.Error(err),
			)
		} else {
			return nil
		}
	}

	return NewValidationError(ErrDeleteMessageFailed, numberRepetion)
}

func (b *Bot) DeleteMessage(deleteMsg tgbotapi.DeleteMessageConfig) error {
	b.limiter.CheckAPI()

	_, err := b.BotAPI.Request(deleteMsg)
	if err != nil {
		return err
	}

	return err
}

func (b *Bot) SendMediaGroup(mediaGroup tgbotapi.MediaGroupConfig) ([]tgbotapi.Message, error) {
	b.limiter.CheckMessage(mediaGroup.ChatID)

	messages, err := b.BotAPI.SendMediaGroup(mediaGroup)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (b *Bot) SendPhoto(photo tgbotapi.PhotoConfig) (tgbotapi.Message, error) {
	b.limiter.CheckMessage(photo.ChatID)

	message, err := b.BotAPI.Send(photo)
	if err != nil {
		return message, err
	}

	return message, nil
}

// EditMessageCaption редактирует подпись к сообщению с учетом ограничений API
func (b *Bot) EditMessageCaption(editMsg tgbotapi.EditMessageCaptionConfig) (*tgbotapi.APIResponse, error) {
	b.limiter.CheckAPI()

	response, err := b.BotAPI.Request(editMsg)
	if err != nil {
		return nil, err
	}

	return response, nil
}
