package bot

import (
	"context"
	"encoding/json"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"github.com/maxaizer/hh-parser/internal/logger"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const addSearchCommandName = "Добавить автопоиск"

type addSearchCommand struct {
	api                  apiInterface
	chatID               int64
	searches             searchRepository
	regions              regionRepository
	inputHandlers        []inputHandler
	curHandlerIndex      int
	searchText           string
	experience           models.Experience
	regionID             string
	schedules            []models.Schedule
	wish                 string
	initialSearchPeriod  int
	finishCallback       func()
	finalMessageKeyboard *botApi.ReplyKeyboardMarkup
}

func newAddSearchCommand(api apiInterface, chatID int64, userRepo searchRepository,
	regionRepo regionRepository) *addSearchCommand {

	cmd := &addSearchCommand{api: api, chatID: chatID, searches: userRepo, regions: regionRepo}

	keywords := newKeywordsInput(chatID, func(keywords string) {
		cmd.searchText = keywords
		cmd.curHandlerIndex++
	})

	experience := newExperienceInput(chatID, func(experience models.Experience) {
		cmd.experience = experience
		cmd.curHandlerIndex++
	})

	region := newRegionInput(chatID, regionRepo, func(regionID string) {
		cmd.regionID = regionID
		cmd.curHandlerIndex++
	})

	schedule := newScheduleInput(chatID, func(schedules []models.Schedule) {
		cmd.schedules = schedules
		cmd.curHandlerIndex++
	})

	wish := newWishInput(chatID, func(wish string) { cmd.wish = wish; cmd.curHandlerIndex++ })
	initialSearchPeriod := newInitialSearchPeriodInput(chatID, func(input string) {
		cmd.initialSearchPeriod, _ = strconv.Atoi(input)
		cmd.curHandlerIndex++
	})

	cmd.inputHandlers = []inputHandler{keywords, experience, region, schedule, wish, initialSearchPeriod}
	return cmd
}

func (c *addSearchCommand) WithFinishCallback(callback func()) {
	c.finishCallback = callback
}

func (c *addSearchCommand) WithKeyboardOnFinalMessage(keyboard botApi.ReplyKeyboardMarkup) {
	c.finalMessageKeyboard = &keyboard
}

func (c *addSearchCommand) SaveState() ([]byte, error) {

	type Alias addSearchCommand
	return json.Marshal(&struct {
		CurHandlerIndex     int
		SearchText          string
		Experience          models.Experience
		RegionID            string
		Schedules           []models.Schedule
		Wish                string
		InitialSearchPeriod int
		*Alias
	}{
		CurHandlerIndex:     c.curHandlerIndex,
		SearchText:          c.searchText,
		Experience:          c.experience,
		RegionID:            c.regionID,
		Schedules:           c.schedules,
		Wish:                c.wish,
		InitialSearchPeriod: c.initialSearchPeriod,
		Alias:               (*Alias)(c),
	})
}

func (c *addSearchCommand) LoadState(data []byte) error {

	type Alias addSearchCommand
	aux := &struct {
		CurHandlerIndex     int
		SearchText          string
		Experience          models.Experience
		RegionID            string
		Schedules           []models.Schedule
		Wish                string
		InitialSearchPeriod int
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.curHandlerIndex = aux.CurHandlerIndex
	c.searchText = aux.SearchText
	c.experience = aux.Experience
	c.regionID = aux.RegionID
	c.schedules = aux.Schedules
	c.wish = aux.Wish
	c.initialSearchPeriod = aux.InitialSearchPeriod
	return nil
}

func (c *addSearchCommand) Run() {
	_, _ = sendWithLogError(c.api, c.inputHandlers[0].InitMessage())
}

func (c *addSearchCommand) OnUserInput(input string) {

	previousIndex := c.curHandlerIndex
	msg := c.inputHandlers[c.curHandlerIndex].HandleInput(input)

	handlerChanged := previousIndex != c.curHandlerIndex
	allHandlersFinished := c.curHandlerIndex >= len(c.inputHandlers)

	if !handlerChanged {
		_, _ = sendWithLogError(c.api, msg)
		return
	}

	if !allHandlersFinished {
		_, _ = sendWithLogError(c.api, c.inputHandlers[c.curHandlerIndex].InitMessage())
		return
	}

	c.addSearch()
	if c.finishCallback != nil {
		c.finishCallback()
	}
}

func (c *addSearchCommand) addSearch() {

	search := models.NewJobSearch(c.chatID, c.searchText, c.regionID, c.experience, c.schedules, c.wish, c.initialSearchPeriod)
	msg := botApi.NewMessage(c.chatID, "")
	if c.finalMessageKeyboard != nil {
		msg.ReplyMarkup = c.finalMessageKeyboard
	}

	if err := c.searches.Add(context.Background(), *search); err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Error(err)
		msg.Text = "Внутренняя ошибка!"
		_, _ = sendWithLogError(c.api, msg)
		return
	}

	msg.Text = "Поиск успешно добавлен!"
	_, _ = sendWithLogError(c.api, msg)
}

func newKeywordsInput(chatID int64, onFinish func(input string)) *textInput {
	return newTextInput(chatID, "Введите ключевые слова для поиска. Например, \"Промывайщик полов\", "+
		"\"Go OR Golang\".", onFinish)
}

func newWishInput(chatID int64, onFinish func(input string)) *textInput {
	return newTextInput(chatID, "Укажите пожелания к вакансии в свободной форме.\n"+
		"Например, \"чтобы соответствовала C# backend разработчику\" или \"хочу вкусняшки в офисе\"", onFinish)
}

func newInitialSearchPeriodInput(chatID int64, onFinish func(input string)) *textInput {
	input := newTextInput(chatID, "Укажите, вакансии за сколько последних дней включить в поиск (от 0 до 5)",
		onFinish)
	input.AddValidation(validation{
		function: func(input string) bool {
			digit, err := strconv.Atoi(input)
			return err == nil && digit >= 0 && digit <= 5
		},
		errorMessage: "Введите число от 0 до 5",
	})
	return input
}
