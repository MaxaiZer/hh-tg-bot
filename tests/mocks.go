package tests

import (
	"context"
	"errors"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"sync"
	"time"
)

type mockVacanciesRetriever struct {
	vacancies []models.Vacancy
}

func (m mockVacanciesRetriever) GetVacancies(search *models.JobSearch, dateFrom time.Time, page, pageSize int) ([]models.Vacancy, error) {
	total := len(m.vacancies)

	start := page * pageSize
	if start >= total {
		return []models.Vacancy{}, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	return m.vacancies[start:end], nil
}

func (m mockVacanciesRetriever) GetVacancy(ID string) (*models.Vacancy, error) {
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

func (m *mockAiService) DoesVacancyMatchSearch(ctx context.Context, search models.JobSearch, vacancy models.Vacancy) (bool, error) {
	time.Sleep(m.responseTime)
	m.mu.Lock()
	defer m.mu.Unlock()

	res := m.responsesQueue[0]
	m.responsesQueue = m.responsesQueue[1:]
	return res.result, res.err
}
