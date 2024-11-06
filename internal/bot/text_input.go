package bot

import botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type validation struct {
	function     func(input string) bool
	errorMessage string
}

type textInput struct {
	chatID      int64
	initMessage string
	onFinish    func(input string)
	validations []validation
}

func newTextInput(chatID int64, initMessage string, onFinish func(input string)) *textInput {
	return &textInput{chatID: chatID, initMessage: initMessage, onFinish: onFinish}
}

func (a *textInput) AddValidation(validation validation) {
	a.validations = append(a.validations, validation)
}

func (a *textInput) InitMessage() botApi.Chattable {
	msg := botApi.NewMessage(a.chatID, a.initMessage)
	msg.ReplyMarkup = keyboardWithExit()
	return msg
}

func (a *textInput) HandleInput(input string) botApi.Chattable {

	for _, _validation := range a.validations {
		if !_validation.function(input) {
			return botApi.NewMessage(a.chatID, _validation.errorMessage)
		}
	}

	a.onFinish(input)
	return nil
}
