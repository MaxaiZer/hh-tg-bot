package bot

import (
	"context"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/logger"
	log "github.com/sirupsen/logrus"
	"strings"
)

type regionInput struct {
	chatID   int64
	onFinish func(regionID string)
	regions  regionRepository
}

func newRegionInput(chatID int64, regionRepo regionRepository, onFinish func(regionID string)) *regionInput {
	return &regionInput{chatID: chatID, regions: regionRepo, onFinish: onFinish}
}

func (a *regionInput) InitMessage() botApi.Chattable {
	msg := botApi.NewMessage(a.chatID, "Введите регион поиска.")
	msg.ReplyMarkup = regionKeyboard()
	return msg
}

func (a *regionInput) HandleInput(input string) botApi.Chattable {

	if input == "Не указывать" {
		a.onFinish("")
		return nil
	}

	regionID, err := a.regions.GetIdByName(context.Background(), strings.ToLower(input))
	if err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Error(err)
		return botApi.NewMessage(a.chatID, "Внутренняя ошибка.")
	}
	if regionID == "" {
		return botApi.NewMessage(a.chatID, "Регион не найден.")
	}

	a.onFinish(regionID)
	return nil
}

func regionKeyboard() botApi.ReplyKeyboardMarkup {
	return botApi.NewReplyKeyboard(
		botApi.NewKeyboardButtonRow(
			botApi.NewKeyboardButton("Москва"),
			botApi.NewKeyboardButton("Санкт-Петербург"),
		),
		botApi.NewKeyboardButtonRow(
			botApi.NewKeyboardButton("Новосибирск"),
			botApi.NewKeyboardButton("Екатеринбург"),
		),
		botApi.NewKeyboardButtonRow(
			botApi.NewKeyboardButton("Не указывать"),
		),
	)
}
