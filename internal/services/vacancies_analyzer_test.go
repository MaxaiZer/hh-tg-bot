package services

import (
	"context"
	"github.com/asaskevich/EventBus"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type mockVacanciesRetriever struct {
	vacancies []entities.Vacancy
}

func (m mockVacanciesRetriever) GetVacancies(search *entities.JobSearch, dateFrom time.Time, page, pageSize int) ([]entities.Vacancy, error) {
	return m.vacancies, nil
}

func (m mockVacanciesRetriever) GetVacancy(ID string) (*entities.Vacancy, error) {
	for _, vacancy := range m.vacancies {
		if vacancy.ID == ID {
			return &vacancy, nil
		}
	}
	return nil, errors.New("not found")
}

type mockAiClient struct {
	mock.Mock
}

func (m *mockAiClient) GenerateResponse(ctx context.Context, request string) (string, error) {
	args := m.Called(ctx, request)
	return args.String(0), args.Error(1)
}

type mockSearches struct {
	mock.Mock
}

func (m *mockSearches) Get(ctx context.Context, pageSize int, pageNum int) ([]entities.JobSearch, error) {
	args := m.Called(ctx, pageSize, pageNum)
	return args.Get(0).([]entities.JobSearch), args.Error(1)
}

func (m *mockSearches) GetByID(ctx context.Context, ID int64) (*entities.JobSearch, error) {
	args := m.Called(ctx, ID)
	return args.Get(0).(*entities.JobSearch), args.Error(1)
}

func (m *mockSearches) UpdateLastCheckedVacancy(ctx context.Context, searchID int, vacancy entities.Vacancy) error {
	return m.Called(ctx, searchID, vacancy).Error(0)
}

type mockVacancies struct {
	mock.Mock
}

func (m *mockVacancies) IsSentToUser(ctx context.Context, vacancy entities.NotifiedVacancyID) (bool, error) {
	args := m.Called(ctx, vacancy)
	return args.Bool(0), args.Error(1)
}

func (m *mockVacancies) RecordAsSentToUser(ctx context.Context, vacancy entities.NotifiedVacancyID) error {
	return m.Called(ctx, vacancy).Error(0)
}

func (m *mockVacancies) AddFailedToAnalyze(ctx context.Context, searchID int, vacancyID string, error string) error {
	return m.Called(ctx, searchID, vacancyID, error).Error(0)
}

func (m *mockVacancies) GetFailedToAnalyze(ctx context.Context) ([]entities.FailedVacancy, error) {
	args := m.Called(ctx)
	failedVacancies, ok := args.Get(0).([]entities.FailedVacancy)
	if !ok {
		return nil, errors.New("type assertion failed for []entities.FailedVacancy")
	}
	return failedVacancies, args.Error(0)
}

func (m *mockVacancies) RemoveFailedToAnalyze(ctx context.Context, maxAttempts int, minUpdateTime time.Time) (int64, error) {
	args := m.Called(ctx, maxAttempts)
	return args.Get(0).(int64), args.Error(1)
}

func Test_AnalyzeVacancy_WhenDuplicateInCache_ShouldIgnore(t *testing.T) {

	ai := mockAiClient{}
	ai.On("GenerateResponse", mock.Anything, mock.Anything).Return("да", nil).Once()
	aiServiceMock := NewAIService(&ai)

	retrieverMock := mockVacanciesRetriever{}

	searches := &mockSearches{}
	search := entities.JobSearch{ID: 1}

	vacancies := &mockVacancies{}
	vacancies.On("IsSentToUser", mock.Anything, mock.Anything).Return(false, nil)
	vacancies.On("RecordAsSentToUser", mock.Anything, mock.Anything).Return(nil)

	vacancy := entities.Vacancy{
		ID:          "1",
		Name:        "Golang developer",
		Description: "test description",
	}

	vacancy2 := entities.Vacancy{
		ID:          "2",
		Name:        "Golang developer",
		Description: "test description",
	}

	analyzer, err := NewVacanciesAnalyzer(EventBus.New(), aiServiceMock, retrieverMock, searches, vacancies, time.Hour)
	assert.NoError(t, err)

	err = analyzer.analyzeVacancyWithAI(context.Background(), vacancy, search)
	assert.NoError(t, err)
	err = analyzer.analyzeVacancyWithAI(context.Background(), vacancy2, search)
	assert.NoError(t, err)
	ai.AssertExpectations(t)
}
