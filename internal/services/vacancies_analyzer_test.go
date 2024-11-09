package services

import (
	"context"
	"github.com/asaskevich/EventBus"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
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

func (m *mockSearches) UpdateLastCheckedVacancy(ctx context.Context, searchID int, vacancy hh.VacancyPreview) error {
	return m.Called(ctx, searchID, vacancy).Error(0)
}

type mockVacancies struct {
	mock.Mock
}

func (m *mockVacancies) IsSentToUser(ctx context.Context, userID int64, vacancyID string) (bool, error) {
	args := m.Called(ctx, userID, vacancyID)
	return args.Bool(0), args.Error(1)
}

func (m *mockVacancies) RecordAsSentToUser(ctx context.Context, userID int64, vacancyID string) error {
	return m.Called(ctx, userID, vacancyID).Error(0)
}

func Test_AnalyzeVacancy_WhenDuplication_ShouldIgnore(t *testing.T) {

	assert := assert.New(t)

	mockBus := EventBus.New()

	ai := mockAiClient{}
	ai.On("GenerateResponse", mock.Anything, mock.Anything).Return("да", nil).Once()

	aiServiceMock := NewAIService(&ai)

	hhMock := hh.NewClient()
	hhMock.SetHTTPClient(&mockHTTPClient{})

	searches := &mockSearches{}
	vacancies := &mockVacancies{}
	vacancies.On("IsSentToUser", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	vacancies.On("RecordAsSentToUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	vacancy := hh.Vacancy{
		VacancyPreview: hh.VacancyPreview{
			ID:   "1",
			Name: "Golang developer",
		},
		Description: "test description",
	}

	vacancy2 := vacancy
	vacancy2.Description = "Super duper golang developer"

	search := entities.JobSearch{ID: 1}

	analyzer, err := NewVacanciesAnalyzer(mockBus, aiServiceMock, hhMock, searches, vacancies)
	assert.NoError(err)

	err = analyzer.analyzeVacancy(context.Background(), vacancy, search)
	assert.NoError(err)
	err = analyzer.analyzeVacancy(context.Background(), vacancy, search)
	assert.NoError(err)
	ai.AssertExpectations(t)

	ai.On("GenerateResponse", mock.Anything, mock.Anything).Return("да", nil).Once()
	err = analyzer.analyzeVacancy(context.Background(), vacancy2, search)
	assert.NoError(err)
	ai.AssertExpectations(t)
}
