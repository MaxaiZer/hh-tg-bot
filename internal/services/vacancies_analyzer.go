package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/asaskevich/EventBus"
	"github.com/maxaizer/hh-parser/internal/entities"
	errs "github.com/maxaizer/hh-parser/internal/errors"
	"github.com/maxaizer/hh-parser/internal/events"
	"github.com/maxaizer/hh-parser/internal/logger"
	"github.com/maxaizer/hh-parser/internal/metrics"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type vacanciesAIService interface {
	DoesVacancyMatchSearch(ctx context.Context, search entities.JobSearch, vacancy entities.Vacancy) (bool, error)
}

type vacanciesRetriever interface {
	GetVacancies(search *entities.JobSearch, dateFrom time.Time, page, pageSize int) ([]entities.Vacancy, error)
	GetVacancy(ID string) (*entities.Vacancy, error)
}

type searchRepository interface {
	Get(ctx context.Context, limit int, offset int) ([]entities.JobSearch, error)
	GetByID(ctx context.Context, ID int64) (*entities.JobSearch, error)
	UpdateLastCheckedVacancy(ctx context.Context, searchID int, vacancy entities.Vacancy) error
}

type vacancyRepository interface {
	IsSentToUser(ctx context.Context, vacancy entities.NotifiedVacancyID) (bool, error)
	RecordAsSentToUser(ctx context.Context, vacancy entities.NotifiedVacancyID) error
	AddFailedToAnalyze(ctx context.Context, searchID int, vacancyID string, error string) error
	RemoveFailedToAnalyze(ctx context.Context, maxAttempts int, minUpdateTime time.Time) (int64, error)
	GetFailedToAnalyze(ctx context.Context) ([]entities.FailedVacancy, error)
}

type analysisRequest struct {
	search  *entities.JobSearch
	vacancy *entities.Vacancy
}

type analysisError struct {
	vacancyID string
	searchID  int
	error     error
}

type VacanciesAnalyzer struct {
	bus                      EventBus.Bus
	searches                 searchRepository
	vacancies                vacancyRepository
	retriever                vacanciesRetriever
	aiService                vacanciesAIService
	lastAnalysisTime         time.Time
	analysisInterval         time.Duration
	searchContexts           sync.Map
	analysisCompleteCallback func()
}

func NewVacanciesAnalyzer(bus EventBus.Bus, aiService vacanciesAIService, vacanciesRetriever vacanciesRetriever,
	searchRepo searchRepository, vacancyRepo vacancyRepository, analysisInterval time.Duration) (*VacanciesAnalyzer, error) {

	v := &VacanciesAnalyzer{
		bus:              bus,
		searches:         searchRepo,
		vacancies:        vacancyRepo,
		retriever:        vacanciesRetriever,
		aiService:        aiService,
		analysisInterval: analysisInterval,
	}

	err := bus.Subscribe(events.SearchDeletedTopic, func(event events.SearchDeleted) {
		v.cancelSearchAnalyze(event.SearchID)
	})
	if err != nil {
		return nil, err
	}

	err = bus.Subscribe(events.SearchEditedTopic, func(event events.SearchEdited) {
		v.cancelSearchAnalyze(event.SearchID)
	})
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (v *VacanciesAnalyzer) WithAnalysisCompleteCallback(f func()) {
	v.analysisCompleteCallback = f
}

func (v *VacanciesAnalyzer) Run() {
	for {
		startTime := time.Now()
		log.Infof("running analysis at %v", time.Now())

		v.runAnalysis()

		executionTime := time.Since(startTime)
		metrics.AnalysisDuration.Observe(executionTime.Seconds())
		log.Infof("analysis ended after %v", executionTime)

		v.rerunAnalysisForFailedVacancies()
		executionTime = time.Now().Sub(startTime.Add(executionTime))
		log.Infof("analysis for failed vacancies ended after %v", executionTime)

		if v.analysisCompleteCallback != nil {
			v.analysisCompleteCallback()
		}

		var sleepTime time.Duration
		if executionTime <= v.analysisInterval {
			sleepTime = v.analysisInterval - executionTime
		} else {
			v.analysisInterval = executionTime + time.Hour
			log.Infof("analysis interval exceeded to %v", v.analysisInterval)
		}

		log.Infof("next analysis time is %v", time.Now().Add(sleepTime))
		time.Sleep(sleepTime)
	}
}

func (v *VacanciesAnalyzer) runAnalysis() {

	errChan := make(chan analysisError, 10)
	errHandler := newErrorHandler(v.vacancies)

	go func() {
		errHandler.Run(errChan)
	}()

	defer func() {
		close(errChan)
		<-errHandler.Done
	}()

	var pageSize, analyzedTotal = 20, 0
	for page := 0; ; page++ {

		jobSearches, err := v.searches.Get(context.Background(), pageSize, page*pageSize)
		if err != nil {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Errorf("failed to get jobSearches: %v", err)
			break
		}
		if len(jobSearches) == 0 {
			break
		}

		var wg sync.WaitGroup
		for _, jobSearch := range jobSearches {
			v.runAnalysisForUserSearch(&wg, errChan, jobSearch)
		}

		wg.Wait()
		analyzedTotal += len(jobSearches)
	}

	log.Infof("handled %v user searches", analyzedTotal)
}

func (v *VacanciesAnalyzer) rerunAnalysisForFailedVacancies() {

	startTime := time.Now().UTC()
	fetchedTotal := 0

	searches := make(map[int]*entities.JobSearch)
	requestChan := make(chan analysisRequest, 10)
	errChan := make(chan analysisError, 10)
	errHandler := newErrorHandler(v.vacancies)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		v.analyzeVacancies(context.Background(), requestChan, errChan)
	}()
	go func() {
		errHandler.Run(errChan)
	}()

	defer func() {
		close(requestChan)
		wg.Wait()
		close(errChan)
		<-errHandler.Done
		log.Infof("fetched total %v failed vacancies", fetchedTotal)

		removed, err := v.vacancies.RemoveFailedToAnalyze(context.Background(), 3, startTime)
		if err != nil {
			log.Errorf("couldn't remove failed vacancies: %v", err)
		} else {
			log.Infof("removed %v old failed vacancies", removed)
		}
	}()

	vacancies, err := v.vacancies.GetFailedToAnalyze(context.Background())
	if err != nil {
		log.Errorf("couldn't get failed analyzed vacancies: %v", err)
		return
	}

	fetchedTotal += len(vacancies)
	for _, vacancyInfo := range vacancies {

		var search *entities.JobSearch
		var ok bool

		if search, ok = searches[vacancyInfo.SearchID]; !ok {
			search, err = v.searches.GetByID(context.Background(), int64(vacancyInfo.SearchID))
			if err != nil {
				log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).
					Errorf("failed to get search by id: %v", err)
				continue
			}
			searches[vacancyInfo.SearchID] = search
		}

		vacancy, err := v.retriever.GetVacancy(vacancyInfo.VacancyID)
		if err != nil {
			log.Errorf("failed to get vacancy by id: %v", err)
			continue
		}

		requestChan <- analysisRequest{search: search, vacancy: vacancy}
	}
}

func (v *VacanciesAnalyzer) runAnalysisForUserSearch(wg *sync.WaitGroup, errChan chan<- analysisError,
	search entities.JobSearch) {

	var dateFrom = search.LastCheckedVacancyTime
	if dateFrom.IsZero() {
		if search.InitialSearchPeriod == 0 {
			dateFrom = search.CreatedAt
		} else {
			dateFrom = time.Now().AddDate(0, 0, -search.InitialSearchPeriod)
		}
	}

	searchCtx, cancel := context.WithCancel(context.Background())
	v.searchContexts.Store(search.ID, cancel)

	wg.Add(1)
	go func(context.Context, entities.JobSearch, time.Time) {
		defer wg.Done()
		defer v.searchContexts.Delete(search.ID)
		v.analyzeVacanciesForSearch(searchCtx, errChan, search, dateFrom)
	}(searchCtx, search, dateFrom)
}

func (v *VacanciesAnalyzer) analyzeVacanciesForSearch(ctx context.Context, errChan chan<- analysisError,
	search entities.JobSearch, dateFrom time.Time) {

	var pageSize, fetchedTotal = 20, 0

	var latestVacancy *entities.Vacancy
	requestChan := make(chan analysisRequest, pageSize)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		v.analyzeVacancies(ctx, requestChan, errChan)
	}()

	for page := 0; ; page++ {

		select {
		case <-ctx.Done():
			log.Infof("analysis canceled for search ID %v", search.ID)
			return
		default:
		}

		vacancies, err := v.retriever.GetVacancies(&search, dateFrom, page, pageSize)
		if err != nil {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeHhApi).Errorf("failed to get vacancies previews: %v", err)
			return //to not update last checked vacancy
		}

		if len(vacancies) == 0 {
			break
		}

		for i := 0; i < len(vacancies); i++ {
			requestChan <- analysisRequest{search: &search, vacancy: &vacancies[i]}
		}

		if latestVacancy == nil {
			latestVacancy = &vacancies[0]
		}
	}

	if latestVacancy != nil {
		//bg because it's better to update last checked in case of cancel
		err := v.searches.UpdateLastCheckedVacancy(context.Background(), search.ID, *latestVacancy)
		if err != nil {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Errorf("failed to update last checked vacancy: %v", err)
		}
	}

	close(requestChan)
	wg.Wait()
	log.Infof("fetched total %v vacancies for search with id %v", fetchedTotal, search.ID)
}

func (v *VacanciesAnalyzer) analyzeVacancies(ctx context.Context, requestChan <-chan analysisRequest, errChan chan<- analysisError) {

	wg := sync.WaitGroup{}

	for {
		select {
		case <-ctx.Done():
			return
		case request, ok := <-requestChan:

			if !ok {
				wg.Wait()
				return
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				err := v.analyzeVacancyWithAI(ctx, *request.vacancy, *request.search)
				if err != nil {
					errChan <- analysisError{request.vacancy.ID, request.search.ID, err}
				} else {
					metrics.HandledVacanciesCounter.Inc()
				}
			}()
		}
	}
}

func (v *VacanciesAnalyzer) analyzeVacancyWithAI(ctx context.Context, vacancy entities.Vacancy, search entities.JobSearch) error {

	vacancyID := createIdForNotifiedVacancy(vacancy, search)
	wasSent, err := v.vacancies.IsSentToUser(ctx, vacancyID)
	if err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).
			Errorf("failed to check if vacancy was sent to user: %v", err)
		return err
	}

	if wasSent {
		return nil
	}

	start := time.Now()
	matched, err := v.aiService.DoesVacancyMatchSearch(ctx, search, vacancy)
	metrics.AnalysisStepDuration.WithLabelValues("ai_analysis").Observe(time.Since(start).Seconds())

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}

	if matched {
		if err = v.handleApproveByAI(ctx, vacancy, search); err != nil {
			return err
		}
	} else {
		metrics.RejectedByAiVacanciesCounter.Inc()
	}
	return nil
}

func (v *VacanciesAnalyzer) handleApproveByAI(ctx context.Context, vacancy entities.Vacancy, search entities.JobSearch) error {

	vacancyID := createIdForNotifiedVacancy(vacancy, search)
	if err := v.vacancies.RecordAsSentToUser(ctx, vacancyID); err != nil {
		if errors.Is(err, errs.VacancyAlreadySentToUser) {
			return nil
		}
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).
			Errorf("failed to record vacancy as send to user: %v", err)
		return err
	}
	event := events.VacancyFound{Search: search, Name: vacancy.Name, Url: vacancy.Url}
	v.bus.Publish(events.VacancyFoundTopic, event)
	return nil
}

func (v *VacanciesAnalyzer) cancelSearchAnalyze(searchID int) {
	if cancel, ok := v.searchContexts.Load(searchID); ok {
		cancel.(context.CancelFunc)()
		v.searchContexts.Delete(searchID)
	}
}

func createIdForNotifiedVacancy(vacancy entities.Vacancy, search entities.JobSearch) entities.NotifiedVacancyID {
	descriptionHash := sha256.Sum256([]byte(vacancy.Description))
	return entities.NotifiedVacancyID{
		UserID:          search.UserID,
		VacancyID:       vacancy.ID,
		DescriptionHash: descriptionHash[:],
	}
}
