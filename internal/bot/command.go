package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/logger"
	log "github.com/sirupsen/logrus"
)

type apiInterface interface {
	Send(chattable tgbotapi.Chattable) (tgbotapi.Message, error)
}

type command interface {
	WithKeyboardOnFinalMessage(tgbotapi.ReplyKeyboardMarkup)
	WithFinishCallback(func())
	Run()
	OnUserInput(input string)
}

type saveable interface {
	SaveState() ([]byte, error)
	LoadState(data []byte) error
}

func sendWithLogError(api apiInterface, chattable tgbotapi.Chattable) (tgbotapi.Message, error) {
	msg, err := api.Send(chattable)
	if err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeTgApi).
			Errorf("error occured while sending message: %v", err)
	}
	return msg, err
}
