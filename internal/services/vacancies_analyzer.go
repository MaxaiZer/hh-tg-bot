package services

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/asaskevich/EventBus"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/events"
	"github.com/maxaizer/hh-parser/internal/logger"
	"github.com/maxaizer/hh-parser/internal/metrics"
	gocache "github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type searchRepository interface {
	Get(ctx context.Context, limit int, offset int) ([]entities.JobSearch, error)
	GetByID(ctx context.Context, ID int64) (*entities.JobSearch, error)
	UpdateLastCheckedVacancy(ctx context.Context, searchID int, vacancy hh.VacancyPreview) error
}

type vacancyRepository interface {
	IsSentToUser(ctx context.Context, userID int64, descriptionHash []byte) (bool, error)
	RecordAsSentToUser(ctx context.Context, userID int64, descriptionHash []byte) error
	AddFailedToAnalyse(ctx context.Context, searchID int, vacancyID string, error string) error
	RemoveFailedToAnalyse(ctx context.Context, maxAttempts int, minUpdateTime time.Time) (int64, error)
	GetFailedToAnalyse(ctx context.Context) ([]entities.FailedVacancy, error)
}

type analysisRequest struct {
	vacancyID string
	search    *entities.JobSearch
}

type analysisError struct {
	vacancyID string
	searchID  int
	error     error
}

type VacanciesAnalyzer struct {
	bus              EventBus.Bus
	searches         searchRepository
	vacancies        vacancyRepository
	hhClient         *hh.Client
	aiService        *AIService
	cacheHelper      *cacheHelper
	lastAnalysisTime time.Time
	analysisInterval time.Duration
	searchContexts   sync.Map
}

func NewVacanciesAnalyzer(bus EventBus.Bus, aiService *AIService, hhClient *hh.Client,
	searchRepo searchRepository, vacancyRepo vacancyRepository) (*VacanciesAnalyzer, error) {

	v := &VacanciesAnalyzer{
		bus:              bus,
		searches:         searchRepo,
		vacancies:        vacancyRepo,
		hhClient:         hhClient,
		aiService:        aiService,
		cacheHelper:      newCacheHelper(hhClient, gocache.New(30*time.Minute, 1*time.Hour)),
		analysisInterval: 3 * time.Hour,
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

	startTime := time.Now()
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
		removed, err := v.vacancies.RemoveFailedToAnalyse(context.Background(), 3, startTime)
		if err != nil {
			log.Errorf("couldn't remove failed vacancies: %v", err)
		} else {
			log.Infof("removed %v old failed vacancies", removed)
		}
	}()

	defer func() {
		close(requestChan)
		wg.Wait()
		close(errChan)
		<-errHandler.Done
		log.Infof("fetched total %v failed vacancies", fetchedTotal)
	}()

	vacancies, err := v.vacancies.GetFailedToAnalyse(context.Background())
	if err != nil {
		log.Errorf("couldn't get failed analysed vacancies: %v", err)
		return
	}

	fetchedTotal += len(vacancies)
	for _, vacancy := range vacancies {

		var search *entities.JobSearch
		var ok bool

		if search, ok = searches[vacancy.SearchID]; !ok {
			search, err = v.searches.GetByID(context.Background(), int64(vacancy.SearchID))
			if err != nil {
				log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).
					Errorf("failed to search by id: %v", err)
				continue
			}
			searches[vacancy.SearchID] = search
		}

		requestChan <- analysisRequest{search: search, vacancyID: vacancy.VacancyID}
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

	var latestVacancy *hh.VacancyPreview
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

		params, err := createHhSearchParams(&search, dateFrom, page, pageSize)
		if err != nil {
			if errors.Is(err, hh.ErrTooDeepPagination) {
				log.Warningf("too deep pagination for search with id %d, page: %d, per page: %d", search.ID, page, pageSize)
				break
			}
			log.Error(err)
			return //to not update latest vacancy
		}

		previews, err := v.hhClient.GetVacancies(*params)
		if err == nil {
			fetchedTotal += len(previews)
		} else {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeHhApi).Errorf("failed to get vacancies previews: %v", err)
			continue
		}

		if len(previews) == 0 {
			break
		}

		for i := 0; i < len(previews); i++ {
			requestChan <- analysisRequest{vacancyID: previews[i].ID, search: &search}
		}

		if latestVacancy == nil {
			latestVacancy = &previews[0]
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
				vacancy, err := v.cacheHelper.getVacancyByID(request.vacancyID)
				if err != nil {
					errChan <- analysisError{vacancy.ID, request.search.ID, err}
					return
				}

				err = v.analyzeVacancyWithAI(ctx, *vacancy, *request.search)
				if err != nil {
					errChan <- analysisError{vacancy.ID, request.search.ID, err}
				} else {
					metrics.HandledVacanciesCounter.Inc()
				}
			}()
		}
	}
}

func (v *VacanciesAnalyzer) analyzeVacancyWithAI(ctx context.Context, vacancy hh.Vacancy, search entities.JobSearch) error {

	if v.cacheHelper.isCachedForSearch(search.ID, vacancy) { //if vacancy with this description already analyzed for this search
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

	v.cacheHelper.cacheForSearch(search.ID, vacancy)
	return nil
}

func (v *VacanciesAnalyzer) handleApproveByAI(ctx context.Context, vacancy hh.Vacancy, search entities.JobSearch) error {

	descriptionHash := sha256.Sum256([]byte(vacancy.Description))
	wasSent, err := v.vacancies.IsSentToUser(ctx, search.UserID, descriptionHash[:])
	if err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).
			Errorf("failed to check if vacancy was sent to user: %v", err)
		return err
	}

	if wasSent {
		return nil
	}

	if err = v.vacancies.RecordAsSentToUser(ctx, search.UserID, descriptionHash[:]); err != nil {
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

func createHhSearchParams(search *entities.JobSearch, dateFrom time.Time, page, pageSize int) (*hh.SearchParameters, error) {
	var err error
	schedules := lo.Map(search.SchedulesAsArray(), func(s entities.Schedule, _ int) hh.Schedule {
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
