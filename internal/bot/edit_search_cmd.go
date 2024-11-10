package bot

import (
	"context"
	"github.com/asaskevich/EventBus"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/events"
	"github.com/maxaizer/hh-parser/internal/logger"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const editSearchCommandName = "Изменить автопоиск"

const (
	inputSearchStep = iota
	inputFieldToEditStep
	inputFieldValueStep
)

type editSearchCommand struct {
	api                  apiInterface
	chatID               int64
	bus                  EventBus.Bus
	searches             searchRepository
	curInput             inputHandler
	curStep              int
	search               *entities.JobSearch
	searchInputFinished  bool
	finishCallback       func()
	finalMessageKeyboard *botApi.ReplyKeyboardMarkup
}

func newEditSearchCommand(api apiInterface, chatID int64, bus EventBus.Bus, searchRepo searchRepository) *editSearchCommand {

	cmd := editSearchCommand{api: api, chatID: chatID, bus: bus, searches: searchRepo, curStep: inputSearchStep}
	input := newSearchInput(chatID, searchRepo, func(s *entities.JobSearch) {
		cmd.search = s
		cmd.curStep = inputFieldToEditStep
	})
	cmd.curInput = input
	return &cmd
}

func (c *editSearchCommand) WithFinishCallback(callback func()) {
	c.finishCallback = callback
}

func (c *editSearchCommand) WithKeyboardOnFinalMessage(keyboard botApi.ReplyKeyboardMarkup) {
	c.finalMessageKeyboard = &keyboard
}

func (c *editSearchCommand) Run() {
	_, _ = sendWithLogError(c.api, c.curInput.InitMessage())
}

func (c *editSearchCommand) OnUserInput(input string) {

	previousStep := c.curStep
	msg := c.curInput.HandleInput(input)

	if c.curStep == previousStep {
		_, _ = sendWithLogError(c.api, msg)
	}

	if c.curStep == inputFieldToEditStep {
		c.curInput = newInputHandlerChoose(c.chatID, func(input string) {
			c.createSearchEditHandler(input)
			c.curStep = inputFieldValueStep
		})
		_, _ = sendWithLogError(c.api, c.curInput.InitMessage())
	}

	if c.curStep == inputFieldValueStep {
		_, _ = sendWithLogError(c.api, c.curInput.InitMessage())
	}
}

func (c *editSearchCommand) createSearchEditHandler(id string) {
	switch id {
	case "0":
		c.curInput = newKeywordsInput(c.chatID, func(input string) {
			c.search.SearchText = input
			c.editSearch()
			c.curStep = inputFieldToEditStep
		})
	case "1":
		c.curInput = newWishInput(c.chatID, func(input string) {
			c.search.UserWish = input
			c.editSearch()
			c.curStep = inputFieldToEditStep
		})
	default:
		log.Errorf("editSearchCommand: wrong search edit handler id")
	}
}

func (c *editSearchCommand) editSearch() {
	if err := c.searches.Update(context.Background(), *c.search); err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Error(err)
		_, _ = sendWithLogError(c.api, botApi.NewMessage(c.chatID, "Внутренняя ошибка!"))
		return
	}

	c.bus.Publish(events.SearchEditedTopic, events.SearchEdited{SearchID: c.search.ID})
	_, _ = sendWithLogError(c.api, botApi.NewMessage(c.chatID, "Поиск успешно обновлён!"))
}

func newInputHandlerChoose(chatID int64, onFinish func(input string)) *textInput {
	input := newTextInput(chatID, "0 - изменить ключевые слова\n1 - изменить пожелание к вакансии.",
		onFinish)
	input.AddValidation(validation{
		function: func(input string) bool {
			digit, err := strconv.Atoi(input)
			return err == nil && digit >= 0 && digit <= 1
		},
		errorMessage: "Введите число от 0 до 1",
	})
	return input
}
