package tests

import (
	"context"
	"errors"
	"github.com/maxaizer/hh-parser/internal/entities"
	"sync"
	"time"
)

type mockVacanciesRetriever struct {
	vacancies []entities.Vacancy
}

func (m mockVacanciesRetriever) GetVacancies(search *entities.JobSearch, dateFrom time.Time, page, pageSize int) ([]entities.Vacancy, error) {
	total := len(m.vacancies)

	start := page * pageSize
	if start >= total {
		return []entities.Vacancy{}, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return m.vacancies[start:end], nil
}

func (m mockVacanciesRetriever) GetVacancy(ID string) (*entities.Vacancy, error) {
	for _, vacancy := range m.vacancies {
		if vacancy.ID == ID {
			return &vacancy, nil
		}
	}
	return nil, errors.New("not found")
}

type mockAiService struct {
	mu             sync.Mutex
	responseTime   time.Duration
	responsesQueue []struct {
		result bool
		err    error
	}
}

func (m *mockAiService) DoesVacancyMatchSearch(ctx context.Context, search entities.JobSearch, vacancy entities.Vacancy) (bool, error) {
	time.Sleep(m.responseTime)
	m.mu.Lock()
	defer m.mu.Unlock()

	res := m.responsesQueue[0]
	m.responsesQueue = m.responsesQueue[1:]
	return res.result, res.err
}
