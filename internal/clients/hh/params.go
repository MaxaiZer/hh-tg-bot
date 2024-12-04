package hh

import (
	"fmt"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/pkg/errors"
	"net/url"
	"strconv"
	"time"
)

var ErrTooDeepPagination = errors.New("too deep pagination")

type Experience string

const (
	NoExperience Experience = "noExperience"
	Between1and3 Experience = "between1And3"
	Between3and6 Experience = "between3And6"
	MoreThan6    Experience = "moreThan6"
)

func ExperienceFrom(experience entities.Experience) (Experience, error) {
	switch experience {
	case entities.NoExperience:
		return NoExperience, nil
	case entities.Between1and3:
		return Between1and3, nil
	case entities.Between3and6:
		return Between3and6, nil
	case entities.MoreThan6:
		return MoreThan6, nil
	default:
		return "", fmt.Errorf("invalid experience type: %v", experience)
	}
}

type Schedule string

const (
	FullDay  Schedule = "fullDay"
	Flexible Schedule = "flexible"
	Remote   Schedule = "remote"
)

func ScheduleFrom(schedule entities.Schedule) (Schedule, error) {
	switch schedule {
	case entities.FullDay:
		return FullDay, nil
	case entities.Flexible:
		return Flexible, nil
	case entities.Remote:
		return Remote, nil
	default:
		return "", fmt.Errorf("invalid schedule type: %v", schedule)
	}
}

type SearchParameters struct {
	Text                   string
	AreaID                 string
	Experience             Experience
	Schedules              []Schedule
	OrderByPublicationTime bool
	DateFrom               time.Time
	Period                 int
	Page                   int
	PerPage                int
}

func (s SearchParameters) Validate() error {

	if s.Period != 0 && !s.DateFrom.IsZero() {
		return fmt.Errorf("can't use both period and dateFrom")
	}

	if s.Page < 0 {
		return fmt.Errorf("page must be non-negative")
	}

	if s.PerPage < 0 || s.PerPage > 100 {
		return fmt.Errorf("per page must be between 0 and 100")
	}

	maxResults := 2000
	maxPage := maxResults / s.PerPage
	if s.Page >= maxPage {
		return ErrTooDeepPagination
	}

	return nil
}

func (s SearchParameters) ToUrlParams() url.Values {

	params := url.Values{}
	params.Add("text", s.Text)
	params.Add("experience", string(s.Experience))
	for _, schedule := range s.Schedules {
		params.Add("schedule", string(schedule))
	}

	if s.AreaID != "" {
		params.Add("area", s.AreaID)
	}

	params.Add("page", strconv.Itoa(s.Page))
	params.Add("perPage", strconv.Itoa(s.PerPage))

	if s.OrderByPublicationTime {
		params.Add("order_by", "publication_time")
	}

	if s.Period != 0 {
		params.Add("period", strconv.Itoa(s.Period))
	}

	if !s.DateFrom.IsZero() {
		params.Add("date_from", s.DateFrom.Format("2006-01-02T15:04:05-0700"))
	}

	return params
}
