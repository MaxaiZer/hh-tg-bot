package bot

import (
	"context"
	"github.com/asaskevich/EventBus"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/events"
	"github.com/maxaizer/hh-parser/internal/logger"
	log "github.com/sirupsen/logrus"
)

const removeSearchCommandName = "Удалить автопоиск"

type removeSearchCommand struct {
	api                  apiInterface
	chatID               int64
	bus                  EventBus.Bus
	searches             searchRepository
	input                inputHandler
	search               *entities.JobSearch
	searchInputFinished  bool
	finishCallback       func()
	finalMessageKeyboard *botApi.ReplyKeyboardMarkup
}

func newRemoveSearchCommand(api apiInterface, chatID int64, bus EventBus.Bus, searchRepo searchRepository) *removeSearchCommand {

	cmd := removeSearchCommand{api: api, chatID: chatID, bus: bus, searches: searchRepo}
	input := newSearchInput(chatID, searchRepo, func(s *entities.JobSearch) {
		cmd.search = s
		cmd.searchInputFinished = true
	})
	cmd.input = input
	return &cmd
}

func (c *removeSearchCommand) WithFinishCallback(callback func()) {
	c.finishCallback = callback
}

func (c *removeSearchCommand) WithKeyboardOnFinalMessage(keyboard botApi.ReplyKeyboardMarkup) {
	c.finalMessageKeyboard = &keyboard
}

func (c *removeSearchCommand) Run() {
	_, _ = sendWithLogError(c.api, c.input.InitMessage())
}

func (c *removeSearchCommand) OnUserInput(input string) {

	msg := c.input.HandleInput(input)

	if !c.searchInputFinished {
		_, _ = sendWithLogError(c.api, msg)
		return
	}

	if c.search != nil {
		c.removeSearch(c.search.ID)
	}

	if c.finishCallback != nil {
		c.finishCallback()
	}
}

func (c *removeSearchCommand) removeSearch(searchID int) {

	msg := botApi.NewMessage(c.chatID, "")
	if c.finalMessageKeyboard != nil {
		msg.ReplyMarkup = c.finalMessageKeyboard
	}

	if err := c.searches.Remove(context.Background(), searchID); err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Error(err)
		msg.Text = "Внутренняя ошибка!"
		_, _ = sendWithLogError(c.api, msg)
		return
	}

	c.bus.Publish(events.SearchDeletedTopic, events.SearchDeleted{SearchID: searchID})
	msg.Text = "Поиск успешно удалён!"
	_, _ = sendWithLogError(c.api, msg)
}
