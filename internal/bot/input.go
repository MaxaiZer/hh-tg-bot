package bot

import botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type inputHandler interface {
	InitMessage() botApi.Chattable
	HandleInput(input string) botApi.Chattable
}
