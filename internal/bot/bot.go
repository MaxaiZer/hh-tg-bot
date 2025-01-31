package bot

import (
	"context"
	"encoding/json"
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

type Repositories struct {
	Search searchRepository
	Region regionRepository
	Data   dataRepository
}

type dataRepository interface {
	Save(ctx context.Context, id string, data []byte) error
	LoadAndRemove(ctx context.Context, id string) ([]byte, error)
}

type searchRepository interface {
	GetByUser(ctx context.Context, userID int64) ([]entities.JobSearch, error)
	GetByID(ctx context.Context, ID int64) (*entities.JobSearch, error)
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
	repositories Repositories
}

const backToMenuCommandName = "В главное меню"

var globalCommands = []string{addSearchCommandName, removeSearchCommandName, backToMenuCommandName, editSearchCommandName}

func NewBot(token string, bus EventBus.Bus, repositories Repositories) (*Bot, error) {

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

	if repositories.Search == nil {
		return nil, errors.New("search repository is nil")
	}

	if repositories.Region == nil {
		return nil, errors.New("region repository is nil")
	}

	if repositories.Data == nil {
		return nil, errors.New("data repository is nil")
	}

	createdBot := &Bot{api: api, userContexts: make(map[int64]*userContext), bus: bus, repositories: repositories}

	err = bus.Subscribe(events.VacancyFoundTopic, createdBot.onVacancyFound)
	if err != nil {
		return nil, err
	}
	return createdBot, nil
}

func (b *Bot) Run() {

	err := b.loadUserContexts()
	if err != nil {
		log.Errorf("Error loading user contexts: %v", err)
	}

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

func (b *Bot) Stop() {
	err := b.saveUserContexts()
	if err != nil {
		log.Errorf("Error saving user contexts: %v", err)
	}
}

func (b *Bot) handleMessage(message *botApi.Message) {

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
	var err error

	if b.userContexts[user.ID] == nil {
		b.userContexts[user.ID] = newUserContext(chat.ID)
	}
	var ctx = b.userContexts[user.ID]

	switch command {
	case "start":
		messageResponse := botApi.NewMessage(chat.ID, "Саламчик попаламчик, родной!")
		messageResponse.ReplyMarkup = defaultReplyKeyboard()
		response = messageResponse
		delete(b.userContexts, user.ID)
	case addSearchCommandName, removeSearchCommandName, editSearchCommandName:
		cmd, cmdErr := b.createCommand(command, user.ID)
		if cmdErr != nil {
			err = fmt.Errorf("couldn't create %s: %w", cmd, cmdErr)
		} else {
			ctx.RunCommand(cmd, command)
		}
	case backToMenuCommandName:
		messageResponse := botApi.NewMessage(chat.ID, "Вы были успешно перенесены в главное меню")
		messageResponse.ReplyMarkup = defaultReplyKeyboard()
		response = messageResponse
		delete(b.userContexts, user.ID)
	default:
		response = botApi.NewMessage(chat.ID, "Неизвестная команда!")
	}

	if err != nil {
		if errors.Is(err, errorNoUserSearches) {
			response = botApi.NewMessage(chat.ID, "У вас нет ни одного автопоиска")
		} else {
			response = botApi.NewMessage(chat.ID, "Внутренняя ошибка!")
			log.Error(err)
		}
	}

	if response == nil {
		return
	}

	_, _ = sendWithLogError(b.api, response)
}

func (b *Bot) createCommand(name string, chatID int64) (command, error) {

	switch name {
	case addSearchCommandName:
		return newAddSearchCommand(b.api, chatID, b.repositories.Search, b.repositories.Region), nil
	case removeSearchCommandName:
		return newRemoveSearchCommand(b.api, chatID, b.bus, b.repositories.Search)
	case editSearchCommandName:
		return newEditSearchCommand(b.api, chatID, b.bus, b.repositories.Search)
	default:
		return nil, fmt.Errorf("unknown command: %v", name)
	}
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

func (b *Bot) saveUserContexts() error {
	data, err := json.Marshal(b.userContexts)
	if err != nil {
		return err
	}
	return b.repositories.Data.Save(context.Background(), "user_contexts", data)
}

func (b *Bot) loadUserContexts() error {
	data, err := b.repositories.Data.LoadAndRemove(context.Background(), "user_contexts")
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &b.userContexts); err != nil {
		return err
	}

	var errs []error
	for i, ctx := range b.userContexts {

		if ctx.curCommandName == "" {
			continue
		}

		cmd, err := b.createCommand(ctx.curCommandName, ctx.chatID)
		if err != nil {
			errs = append(errs, err)
			delete(b.userContexts, i)
			continue
		}

		saveableCmd, ok := cmd.(saveable)
		if !ok {
			ctx.ResumeCommandAfterBotRestart(cmd)
			continue
		}

		err = saveableCmd.LoadState(ctx.curCommandState)
		if err != nil {
			errs = append(errs, err)
			delete(b.userContexts, i)
			continue
		}

		ctx.ResumeCommandAfterBotRestart(cmd)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
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
