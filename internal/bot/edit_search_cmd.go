package bot

import (
	"context"
	"encoding/json"
	"fmt"
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
	inputKeywordsStep
	inputWishStep
)

type editSearchCommand struct {
	api                  apiInterface
	chatID               int64
	bus                  EventBus.Bus
	searches             searchRepository
	inputHandlers        [4]inputHandler
	curInputIdx          int
	search               *entities.JobSearch
	finishCallback       func()
	finalMessageKeyboard *botApi.ReplyKeyboardMarkup
}

func newEditSearchCommand(api apiInterface, chatID int64, bus EventBus.Bus, searchRepo searchRepository) (*editSearchCommand, error) {

	cmd := editSearchCommand{api: api, chatID: chatID, bus: bus, searches: searchRepo, curInputIdx: inputSearchStep}

	var err error
	cmd.inputHandlers[inputSearchStep], err = newSearchInput(chatID, searchRepo, func(s *entities.JobSearch) {
		cmd.search = s
		cmd.curInputIdx = inputFieldToEditStep
	})
	cmd.inputHandlers[inputFieldToEditStep] = newInputHandlerChoose(cmd.chatID, func(input string) {
		num, _ := strconv.Atoi(input)
		switch num {
		case 0:
			cmd.curInputIdx = inputKeywordsStep
		case 1:
			cmd.curInputIdx = inputWishStep
		default:
			log.Errorf("editSearchCommand: wrong handler number: %d", num)
			_, _ = sendWithLogError(cmd.api, botApi.NewMessage(cmd.chatID, "Внутренняя ошибка"))
			cmd.finishCallback()
		}
	})
	cmd.inputHandlers[inputKeywordsStep] = newKeywordsInput(cmd.chatID, func(input string) {
		cmd.search.SearchText = input
		cmd.editSearch()
		cmd.curInputIdx = inputFieldToEditStep
	})
	cmd.inputHandlers[inputWishStep] = newWishInput(cmd.chatID, func(input string) {
		cmd.search.UserWish = input
		cmd.editSearch()
		cmd.curInputIdx = inputFieldToEditStep
	})

	return &cmd, err
}

func (c *editSearchCommand) WithFinishCallback(callback func()) {
	c.finishCallback = callback
}

func (c *editSearchCommand) WithKeyboardOnFinalMessage(keyboard botApi.ReplyKeyboardMarkup) {
	c.finalMessageKeyboard = &keyboard
}

func (c *editSearchCommand) SaveState() ([]byte, error) {

	searchIdToSave := 0
	if c.search != nil {
		searchIdToSave = c.search.ID
	}

	type Alias editSearchCommand
	return json.Marshal(&struct {
		SearchInputFinished bool
		SearchID            int
		CurInputIdx         int
		*Alias
	}{
		SearchInputFinished: c.search != nil,
		SearchID:            searchIdToSave,
		CurInputIdx:         c.curInputIdx,
	})
}

func (c *editSearchCommand) LoadState(data []byte) error {

	type Alias editSearchCommand
	aux := &struct {
		SearchInputFinished bool
		SearchID            int
		CurInputIdx         int
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.curInputIdx = aux.CurInputIdx
	if !aux.SearchInputFinished {
		return nil
	}

	search, err := c.searches.GetByID(context.Background(), int64(aux.SearchID))
	if err != nil {
		return fmt.Errorf("couldn't fetch user search: %w", err)
	}

	c.search = search
	return nil
}

func (c *editSearchCommand) Run() {
	_, _ = sendWithLogError(c.api, c.inputHandlers[c.curInputIdx].InitMessage())
}

func (c *editSearchCommand) OnUserInput(input string) {

	previousIdx := c.curInputIdx
	msg := c.inputHandlers[c.curInputIdx].HandleInput(input)

	if c.curInputIdx == previousIdx {
		_, _ = sendWithLogError(c.api, msg)
		return
	}

	_, _ = sendWithLogError(c.api, c.inputHandlers[c.curInputIdx].InitMessage())
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
