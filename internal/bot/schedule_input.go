package bot

import (
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/entities"
	"strconv"
	"strings"
)

type scheduleInput struct {
	chatID   int64
	onFinish func(schedules []entities.Schedule)
}

func newScheduleInput(chatID int64, onFinish func(schedules []entities.Schedule)) *scheduleInput {
	return &scheduleInput{chatID: chatID, onFinish: onFinish}
}

func (a *scheduleInput) InitMessage() botApi.Chattable {
	msg := botApi.NewMessage(a.chatID, "Введите желаемый график работы.\n"+
		"0 - без разницы, 1 - полный день, 2 - гибкий график, 3 - удалённая работа\n"+
		"также можно комбинировать: \"2, 3\"")
	msg.ReplyMarkup = keyboardWithExit()
	return msg
}

func (a *scheduleInput) HandleInput(input string) botApi.Chattable {

	schedules := strings.Split(input, ",")

	if input == "0" {
		a.onFinish(nil)
		return nil
	}

	for i := 0; i < len(schedules); i++ {
		schedules[i] = strings.TrimSpace(schedules[i])
		_, err := strconv.Atoi(schedules[i])
		if schedules[i] == "" || err != nil {
			return botApi.NewMessage(a.chatID, "Неверный ввод.")
		}
	}

	var res []entities.Schedule

	for i := 0; i < len(schedules); i++ {
		switch schedules[i] {
		case "1":
			res = append(res, entities.FullDay)
		case "2":
			res = append(res, entities.Flexible)
		case "3":
			res = append(res, entities.Remote)
		default:
			return botApi.NewMessage(a.chatID, "Неверный ввод.")
		}
	}

	a.onFinish(res)
	return nil
}
