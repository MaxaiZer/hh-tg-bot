package bot

import (
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/entities"
)

type experienceLevel string

const (
	noExperience experienceLevel = "Нет опыта 😡"
	between1and3 experienceLevel = "Между 1 и 3"
	between3and6 experienceLevel = "Между 3 и 6"
	moreThan6    experienceLevel = "Больше 6 лет"
)

type experienceInput struct {
	chatID   int64
	onFinish func(experience entities.Experience)
}

func newExperienceInput(chatID int64, onFinish func(experience entities.Experience)) *experienceInput {
	return &experienceInput{chatID: chatID, onFinish: onFinish}
}

func (a *experienceInput) InitMessage() botApi.Chattable {
	msg := botApi.NewMessage(a.chatID, "Введите опыт работы.")
	msg.ReplyMarkup = experienceKeyboard()
	return msg
}

func (a *experienceInput) HandleInput(input string) botApi.Chattable {

	var experience entities.Experience

	switch experienceLevel(input) {
	case noExperience:
		experience = entities.NoExperience
	case between1and3:
		experience = entities.Between1and3
	case between3and6:
		experience = entities.Between3and6
	case moreThan6:
		experience = entities.MoreThan6
	default:
		return botApi.NewMessage(a.chatID, "Неправильный ввод 😔.")
	}

	a.onFinish(experience)
	return nil
}

func experienceKeyboard() botApi.ReplyKeyboardMarkup {
	return botApi.NewReplyKeyboard(
		botApi.NewKeyboardButtonRow(
			botApi.NewKeyboardButton(string(noExperience)),
			botApi.NewKeyboardButton(string(between1and3)),
		),
		botApi.NewKeyboardButtonRow(
			botApi.NewKeyboardButton(string(between3and6)),
			botApi.NewKeyboardButton(string(moreThan6)),
		))
}
