package tests

import (
	"context"
	"github.com/asaskevich/EventBus"
	"github.com/maxaizer/hh-parser/internal/domain/events"
	"github.com/maxaizer/hh-parser/internal/domain/models"
	"github.com/maxaizer/hh-parser/internal/repositories"
	"github.com/maxaizer/hh-parser/internal/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var search = models.NewJobSearch(0, "Golang", "0", models.NoExperience,
	[]models.Schedule{models.Remote}, "хочу питсы", 2)

var vacancy = models.Vacancy{
	ID:          "0",
	Url:         "hh.ru/vacancies/0",
	Name:        "Golang developer",
	Description: "раб за копейки",
	KeySkills:   nil,
	PublishedAt: time.Now(),
}

func clearDb() {
	dbCtx.DB.Exec("DELETE from failed_vacancies WHERE TRUE")
	dbCtx.DB.Exec("DELETE from notified_vacancies WHERE TRUE")
}

func Test_Analysis_DuplicatesByDescriptionAreIgnored(t *testing.T) {

	defer clearDb()

	aiServiceMock := mockAiService{
		responsesQueue: []struct {
			result bool
			err    error
		}{
			{result: true, err: nil},
			{result: true, err: nil},
		},
	}

	notifications := 0
	bus := EventBus.New()
	bus.Subscribe(events.VacancyFoundTopic, func(found events.VacancyFound) {
		notifications++
		t.Log(found)
	})

	//same description, different id
	dublicate := vacancy
	dublicate.ID = "10"

	retrieverMock := mockVacanciesRetriever{
		vacancies: []models.Vacancy{vacancy, dublicate},
	}

	searches := repositories.NewSearchRepository(dbCtx.DB)
	vacancies := repositories.NewVacanciesRepository(dbCtx.DB)

	analyzer, err := services.NewVacanciesAnalyzer(bus, &aiServiceMock, retrieverMock,
		searches, vacancies, time.Hour)
	assert.NoError(t, err)

	analysisComplete := make(chan struct{})

	analyzer.WithAnalysisCompleteCallback(func() {
		analysisComplete <- struct{}{}
	})

	go analyzer.Run()

	select {
	case <-time.After(30 * time.Second):
		assert.Fail(t, "timed out")
	case <-analysisComplete:
	}

	failed, err := vacancies.GetFailedToAnalyze(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, failed)
	assert.Equal(t, 1, notifications)
}

func Test_Analysis_DuplicatesByDescription_HtmlTagsAreIgnored(t *testing.T) {

	defer clearDb()

	aiServiceMock := mockAiService{
		responsesQueue: []struct {
			result bool
			err    error
		}{
			{result: true, err: nil},
			{result: true, err: nil},
		},
	}

	notifications := 0
	bus := EventBus.New()
	bus.Subscribe(events.VacancyFoundTopic, func(found events.VacancyFound) {
		notifications++
		t.Log(found)
	})

	//same description with added html tags + extra spaces, different id
	duplicate := vacancy
	duplicate.ID = "10"
	duplicate.Description = "раб  за      копейки<li></li><strong></strong><p></p><ul></ul><em></em>"

	retrieverMock := mockVacanciesRetriever{
		vacancies: []models.Vacancy{vacancy, duplicate},
	}

	searches := repositories.NewSearchRepository(dbCtx.DB)
	vacancies := repositories.NewVacanciesRepository(dbCtx.DB)

	analyzer, err := services.NewVacanciesAnalyzer(bus, &aiServiceMock, retrieverMock,
		searches, vacancies, time.Hour)
	assert.NoError(t, err)

	analysisComplete := make(chan struct{})

	analyzer.WithAnalysisCompleteCallback(func() {
		analysisComplete <- struct{}{}
	})

	go analyzer.Run()

	select {
	case <-time.After(30 * time.Second):
		assert.Fail(t, "timed out")
	case <-analysisComplete:
	}

	failed, err := vacancies.GetFailedToAnalyze(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, failed)
	assert.Equal(t, 1, notifications)
}

func Test_Analysis_DuplicatesByIdAreIgnored(t *testing.T) {

	defer clearDb()

	aiServiceMock := mockAiService{
		responsesQueue: []struct {
			result bool
			err    error
		}{
			{result: true, err: nil},
			{result: true, err: nil},
		},
	}

	notifications := 0
	bus := EventBus.New()
	bus.Subscribe(events.VacancyFoundTopic, func(found events.VacancyFound) {
		notifications++
		t.Log(found)
	})

	//description was edited lately
	duplicate := vacancy
	duplicate.Description = "раб за ещё меньшие копейки"

	retrieverMock := mockVacanciesRetriever{
		vacancies: []models.Vacancy{vacancy, duplicate},
	}

	searches := repositories.NewSearchRepository(dbCtx.DB)
	vacancies := repositories.NewVacanciesRepository(dbCtx.DB)

	analyzer, err := services.NewVacanciesAnalyzer(bus, &aiServiceMock, retrieverMock,
		searches, vacancies, time.Hour)
	assert.NoError(t, err)

	analysisComplete := make(chan struct{})

	analyzer.WithAnalysisCompleteCallback(func() {
		analysisComplete <- struct{}{}
	})

	go analyzer.Run()

	select {
	case <-time.After(30 * time.Second):
		assert.Fail(t, "timed out")
	case <-analysisComplete:
	}

	failed, err := vacancies.GetFailedToAnalyze(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, failed)
	assert.Equal(t, 1, notifications)
}

func Test_RerunAnalysisForFailedVacancy_Success(t *testing.T) {

	defer clearDb()

	aiServiceMock := mockAiService{
		responseTime: 100 * time.Millisecond,
		responsesQueue: []struct {
			result bool
			err    error
		}{
			{result: false, err: errors.New("AI error!")},
			{result: true, err: nil},
		},
	}
	retrieverMock := mockVacanciesRetriever{
		vacancies: []models.Vacancy{vacancy},
	}

	searches := repositories.NewSearchRepository(dbCtx.DB)
	vacancies := repositories.NewVacanciesRepository(dbCtx.DB)

	analyzer, err := services.NewVacanciesAnalyzer(EventBus.New(), &aiServiceMock, retrieverMock,
		searches, vacancies, time.Hour)
	assert.NoError(t, err)

	analysisComplete := false

	analyzer.WithAnalysisCompleteCallback(func() {
		analysisComplete = true
	})

	go analyzer.Run()

	select {
	case <-time.After(30 * time.Second):
		assert.Fail(t, "timed out")
	case <-time.After(1 * time.Second):
		if analysisComplete {
			break
		}
	}

	assert.Empty(t, aiServiceMock.responsesQueue)

	failed, err := vacancies.GetFailedToAnalyze(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, failed)
}

func Test_RerunAnalysisForFailedVacancy_FailedAgain(t *testing.T) {

	defer clearDb()

	aiServiceMock := mockAiService{
		responseTime: 100 * time.Millisecond,
		responsesQueue: []struct {
			result bool
			err    error
		}{
			{result: false, err: errors.New("AI error!")},
			{result: false, err: errors.New("AI error!")},
		},
	}
	retrieverMock := mockVacanciesRetriever{
		vacancies: []models.Vacancy{vacancy},
	}

	searches := repositories.NewSearchRepository(dbCtx.DB)
	vacancies := repositories.NewVacanciesRepository(dbCtx.DB)

	analyzer, err := services.NewVacanciesAnalyzer(EventBus.New(), &aiServiceMock, retrieverMock,
		searches, vacancies, time.Hour)
	assert.NoError(t, err)

	analysisComplete := make(chan struct{})

	analyzer.WithAnalysisCompleteCallback(func() {
		analysisComplete <- struct{}{}
	})

	go analyzer.Run()

	select {
	case <-time.After(30 * time.Second):
		assert.Fail(t, "timed out")
	case <-analysisComplete:
	}

	assert.Empty(t, aiServiceMock.responsesQueue)

	failed, err := vacancies.GetFailedToAnalyze(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, failed)
	if len(failed) == 0 {
		return
	}
	assert.Equal(t, vacancy.ID, failed[0].VacancyID)
	assert.Equal(t, search.ID, failed[0].SearchID)
	assert.Equal(t, 2, failed[0].Attempts)
}
