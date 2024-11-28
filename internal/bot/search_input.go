package bot

import (
	"context"
	"fmt"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/logger"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strconv"
)

var errorNoUserSearches = errors.New("user has no searches")

type searchInput struct {
	chatID       int64
	searches     searchRepository
	userSearches []entities.JobSearch
	onFinish     func(search *entities.JobSearch)
}

func newSearchInput(chatID int64, searchRepo searchRepository, onFinish func(search *entities.JobSearch)) (*searchInput, error) {
	userSearches, err := searchRepo.GetByUser(context.Background(), chatID)
	if err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Error(err)
		return nil, err
	}
	if len(userSearches) == 0 {
		return nil, errorNoUserSearches
	}
	return &searchInput{chatID: chatID, searches: searchRepo, userSearches: userSearches, onFinish: onFinish}, nil
}

func (s *searchInput) InitMessage() botApi.Chattable {

	text := "Введите номер поиска:\n"
	text += s.searchesToText(s.userSearches)

	msg := botApi.NewMessage(s.chatID, text)
	msg.ReplyMarkup = keyboardWithExit()
	return msg
}

func (s *searchInput) HandleInput(input string) botApi.Chattable {

	number, err := strconv.Atoi(input)
	if err != nil {
		return botApi.NewMessage(s.chatID, "Введите число!")
	}

	if number < 1 || number > len(s.userSearches) {
		return botApi.NewMessage(s.chatID, "Нет автопоиска с таким номером.")
	}

	s.onFinish(&s.userSearches[number-1])
	return nil
}

func (s *searchInput) searchesToText(searches []entities.JobSearch) (text string) {
	for i := 0; i < len(searches); i++ {

		text += strconv.Itoa(i+1) + ": \"" + searches[i].SearchText + "\""

		if searches[i].RegionID == "" {
			text += ", регион не важен"
		}

		experience, err := experienceToText(searches[i].Experience)
		if err != nil {
			log.Errorf(err.Error())
		} else {
			text += ", " + experience
		}

		if searches[i].Schedules == "" {
			text += ", график работы не важен"
		}

		for _, schedule := range searches[i].SchedulesAsArray() {
			schedule, err := scheduleToText(schedule)
			if err != nil {
				log.Errorf(err.Error())
			} else {
				text += ", " + schedule
			}
		}

		text += ", пожелание: \"" + searches[i].UserWish + "\""

		createdAt := searches[i].CreatedAt.Format("2006-01-02 15:04:05")
		text += ", создан " + createdAt + "\n"
	}
	return text
}

func scheduleToText(schedule entities.Schedule) (string, error) {
	switch schedule {
	case entities.FullDay:
		return "полный день", nil
	case entities.Flexible:
		return "гибкий график", nil
	case entities.Remote:
		return "удалённая работа", nil
	default:
		return "", fmt.Errorf("invalid schedule: %s", schedule)
	}
}

func experienceToText(experience entities.Experience) (string, error) {
	switch experience {
	case entities.NoExperience:
		return "без опыта", nil
	case entities.Between1and3:
		return "опыт от 1 до 3 лет", nil
	case entities.Between3and6:
		return "опыт от 3 до 6 лет", nil
	case entities.MoreThan6:
		return "опыт от 6 лет", nil
	default:
		return "", fmt.Errorf("invalid experience: %s", experience)
	}
}
