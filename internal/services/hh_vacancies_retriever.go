package services

import (
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"time"
)

type HHVacanciesRetriever struct {
	client *hh.Client
}

func NewHHVacanciesRetriever(client *hh.Client) *HHVacanciesRetriever {
	return &HHVacanciesRetriever{client: client}
}

func (r *HHVacanciesRetriever) GetVacancies(search *models.JobSearch, dateFrom time.Time, page, pageSize int) ([]models.Vacancy, error) {

	params, err := createHhSearchParams(search, dateFrom, page, pageSize)
	if err != nil {
		if errors.Is(err, hh.ErrTooDeepPagination) {
			log.Warningf("too deep pagination for search with id %d, page: %d, per page: %d", search.ID, page, pageSize)
			return []models.Vacancy{}, nil
		}
		log.Error(err)
		return nil, err
	}

	previews, err := r.client.GetVacancies(*params)
	if err != nil {
		return nil, err
	}

	var vacancies []models.Vacancy
	for _, preview := range previews {
		vacancy, err := r.GetVacancy(preview.ID)
		if err != nil {
			return nil, err
		}
		vacancies = append(vacancies, *vacancy)
	}

	return vacancies, nil
}

func (r *HHVacanciesRetriever) GetVacancy(ID string) (*models.Vacancy, error) {

	vacancy, err := r.client.GetVacancy(ID)
	if err != nil {
		return nil, err
	}

	var skills []string
	for _, skill := range vacancy.KeySkills {
		skills = append(skills, skill.Name)
	}

	return &models.Vacancy{
		ID:          vacancy.ID,
		Url:         vacancy.Url,
		Name:        vacancy.Name,
		Description: vacancy.Description,
		KeySkills:   skills,
		PublishedAt: vacancy.PublishedAt.Time,
	}, nil
}

func createHhSearchParams(search *models.JobSearch, dateFrom time.Time, page, pageSize int) (*hh.SearchParameters, error) {
	var err error
	schedules := lo.Map(search.SchedulesAsArray(), func(s models.Schedule, _ int) hh.Schedule {
		schedule, _err := hh.ScheduleFrom(s)
		if _err != nil {
			err = _err
		}
		return schedule
	})

	if err != nil {
		return nil, errors.New("error map schedules")
	}

	params := hh.SearchParameters{
		Text:                   search.SearchText,
		Experience:             hh.Experience(search.Experience),
		Schedules:              schedules,
		DateFrom:               dateFrom,
		AreaID:                 search.RegionID,
		OrderByPublicationTime: true,
		Page:                   page,
		PerPage:                pageSize,
	}
	if err = params.Validate(); err != nil {
		return nil, err
	}
	return &params, nil
}
