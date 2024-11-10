package bot

import (
	"context"
	"errors"
	"fmt"
	"github.com/asaskevich/EventBus"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/events"
	"github.com/maxaizer/hh-parser/internal/logger"
	log "github.com/sirupsen/logrus"
	"slices"
)

type searchRepository interface {
	GetByUser(ctx context.Context, userID int64) ([]entities.JobSearch, error)
	Add(ctx context.Context, search entities.JobSearch) error
	Update(ctx context.Context, search entities.JobSearch) error
	Remove(ctx context.Context, ID int) error
}

type regionRepository interface {
	GetIdByName(ctx context.Context, name string) (string, error)
}

type Bot struct {
	api          *botApi.BotAPI
	userContexts map[int64]*userContext
	bus          EventBus.Bus
	searches     searchRepository
	regions      regionRepository
}

const backToMenuCommandName = "В главное меню"

var globalCommands = []string{addSearchCommandName, removeSearchCommandName, backToMenuCommandName, editSearchCommandName}

func NewBot(token string, bus EventBus.Bus, searchRepo searchRepository, regionRepo regionRepository) (*Bot, error) {

	api, err := botApi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Infof("Authorized on account %s", api.Self.UserName)

	err = botApi.SetLogger(log.StandardLogger())
	if err != nil {
		return nil, err
	}

	if bus == nil {
		return nil, errors.New("bus is nil")
	}

	if searchRepo == nil {
		return nil, errors.New("search repository is nil")
	}

	if regionRepo == nil {
		return nil, errors.New("region repository is nil")
	}

	createdBot := &Bot{api: api, userContexts: make(map[int64]*userContext), bus: bus,
		searches: searchRepo, regions: regionRepo}

	err = bus.Subscribe(events.VacancyFoundTopic, createdBot.onVacancyFound)
	if err != nil {
		return nil, err
	}
	return createdBot, nil
}

func (b *Bot) Start() {
	updateConfig := botApi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	for update := range updates {

		if update.Message == nil {
			continue
		}

		if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
			continue
		}

		go b.handleMessage(update.Message)
	}
}

func (b *Bot) handleMessage(message *botApi.Message) {

	log.Infof("message from %s: %s", message.From.UserName, message.Text)

	cmd := message.Command()
	if cmd == "" && slices.Contains(globalCommands, message.Text) {
		cmd = message.Text
	}

	if cmd != "" {
		b.handleCommand(message.From, message.Chat, cmd, message.CommandArguments())
	} else {
		b.handleInput(message.From, message.Chat, message.Text)
	}
	ctx := b.userContexts[message.From.ID]
	if ctx == nil {
		ctx = &userContext{}
	}
}

func (b *Bot) handleCommand(user *botApi.User, chat *botApi.Chat, command string, args string) {

	var response botApi.Chattable

	if b.userContexts[user.ID] == nil {
		b.userContexts[user.ID] = newUserContext(chat.ID)
	}
	var ctx = b.userContexts[user.ID]

	switch command {
	case "start":
		messageResponse := botApi.NewMessage(chat.ID, "Саламчик попаламчик, родной!")
		messageResponse.ReplyMarkup = defaultReplyKeyboard()
		response = messageResponse
	case addSearchCommandName:
		ctx.RunCommand(newAddSearchCommand(b.api, chat.ID, b.searches, b.regions))
	case removeSearchCommandName:
		ctx.RunCommand(newRemoveSearchCommand(b.api, chat.ID, b.bus, b.searches))
	case editSearchCommandName:
		ctx.RunCommand(newEditSearchCommand(b.api, chat.ID, b.bus, b.searches))
	case backToMenuCommandName:
		messageResponse := botApi.NewMessage(chat.ID, "Вы были успешно перенесены в главное меню")
		messageResponse.ReplyMarkup = defaultReplyKeyboard()
		response = messageResponse
	default:
		response = botApi.NewMessage(chat.ID, "Неизвестная команда!")
	}

	if response == nil {
		return
	}

	_, _ = sendWithLogError(b.api, response)
}

func (b *Bot) handleInput(user *botApi.User, chat *botApi.Chat, input string) {

	ctx := b.userContexts[user.ID]
	if ctx == nil {
		return
	}

	var response botApi.Chattable

	if ctx.HasRunningCommand() {
		ctx.OnUserInput(input)
	} else {
		response = botApi.NewMessage(chat.ID, "Ожидается команда.")
	}

	if response == nil {
		return
	}

	_, err := b.api.Send(response)
	if err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeTgApi).Errorf("error occured while sending message: %v", err)
	}
}

func (b *Bot) onVacancyFound(event events.VacancyFound) {
	msg := botApi.NewMessage(event.Search.UserID,
		fmt.Sprintf("Найдена подходящая вакансия по поиску \"%v\":\n%v", event.Search.SearchText, event.Url))
	if _, err := b.api.Send(msg); err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeTgApi).Errorf("error occured while sending message: %v", err)
	}
}

func defaultReplyKeyboard() botApi.ReplyKeyboardMarkup {

	k := botApi.NewReplyKeyboard()
	k.OneTimeKeyboard = true

	return botApi.NewReplyKeyboard(
		botApi.NewKeyboardButtonRow(
			botApi.NewKeyboardButton(addSearchCommandName),
			botApi.NewKeyboardButton(editSearchCommandName),
			botApi.NewKeyboardButton(removeSearchCommandName),
		),
	)
}

func keyboardWithExit() botApi.ReplyKeyboardMarkup {
	return botApi.NewReplyKeyboard(
		botApi.NewKeyboardButtonRow(
			botApi.NewKeyboardButton(backToMenuCommandName),
		),
	)
}
