package models

import (
	"errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Experience string

const (
	NoExperience Experience = "noExperience"
	Between1and3 Experience = "between1And3"
	Between3and6 Experience = "between3And6"
	MoreThan6    Experience = "moreThan6"
)

type Schedule string

const (
	FullDay  Schedule = "fullDay"
	Flexible Schedule = "flexible"
	Remote   Schedule = "remote"
)

func ToSchedule(s string) (Schedule, error) {
	switch s {
	case string(FullDay):
		return FullDay, nil
	case string(Flexible):
		return Flexible, nil
	case string(Remote):
		return Remote, nil
	default:
		return "", errors.New("invalid schedule type")
	}
}

type JobSearch struct {
	ID                     int
	UserID                 int64
	SearchText             string
	Schedules              string
	RegionID               string
	Experience             Experience
	UserWish               string
	InitialSearchPeriod    int
	LastCheckedVacancyTime time.Time
	CreatedAt              time.Time
}

func NewJobSearch(
	userID int64,
	searchText string,
	regionID string,
	experience Experience,
	schedules []Schedule,
	userWish string,
	initialSearchPeriod int,
) *JobSearch {

	schedulesAsStr := lo.Map(schedules, func(item Schedule, _ int) string {
		return string(item)
	})
	return &JobSearch{
		UserID:              userID,
		SearchText:          searchText,
		RegionID:            regionID,
		Experience:          experience,
		Schedules:           strings.Join(schedulesAsStr, ","),
		UserWish:            userWish,
		InitialSearchPeriod: initialSearchPeriod,
	}
}

func (s *JobSearch) SchedulesAsArray() []Schedule {
	if s.Schedules == "" {
		return []Schedule{}
	}

	return lo.Map(strings.Split(s.Schedules, ","), func(item string, _ int) Schedule {
		schedule, err := ToSchedule(item)
		if err != nil {
			log.Error(err)
		}
		return schedule
	})
}
