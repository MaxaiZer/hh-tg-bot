package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/asaskevich/EventBus"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/events"
	"github.com/maxaizer/hh-parser/internal/logger"
	"github.com/maxaizer/hh-parser/internal/metrics"
	gocache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

type searchRepository interface {
	Get(ctx context.Context, pageSize int, pageNum int) ([]entities.JobSearch, error)
	UpdateLastCheckedVacancy(ctx context.Context, searchID int, vacancy hh.VacancyPreview) error
}

type vacancyRepository interface {
	IsSentToUser(ctx context.Context, userID int64, vacancyID string) (bool, error)
	RecordAsSentToUser(ctx context.Context, userID int64, vacancyID string) error
}

type VacanciesAnalyzer struct {
	bus              EventBus.Bus
	searches         searchRepository
	vacancies        vacancyRepository
	hhClient         *hh.Client
	aiService        *AIService
	cache            *gocache.Cache
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
		analysisInterval: 3 * time.Hour,
		cache:            gocache.New(10*time.Minute, 20*time.Minute),
	}
	err := bus.Subscribe(events.SearchDeletedTopic, v.onSearchDeletedEvent)
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

	var pageSize, analyzedTotal = 20, 0

	for pageNum := 1; ; pageNum++ {

		jobSearches, err := v.searches.Get(context.Background(), pageSize, pageNum)
		if err != nil {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Errorf("failed to get jobSearches: %v", err)
			break
		}
		if len(jobSearches) == 0 {
			break
		}

		var wg sync.WaitGroup
		for _, jobSearch := range jobSearches {
			v.runAnalysisForUserSearch(&wg, jobSearch)
		}

		wg.Wait()
		analyzedTotal += len(jobSearches)
	}

	log.Infof("handled %v user searches", analyzedTotal)
}

func (v *VacanciesAnalyzer) runAnalysisForUserSearch(wg *sync.WaitGroup, search entities.JobSearch) {

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
		v.analyzeVacanciesForSearch(searchCtx, search, dateFrom)
	}(searchCtx, search, dateFrom)
}

func (v *VacanciesAnalyzer) analyzeVacanciesForSearch(ctx context.Context, search entities.JobSearch, dateFrom time.Time) {

	var pageSize, fetchedTotal = 20, 0

	schedules, err := tryMapArray(search.SchedulesAsArray(),
		func(s entities.Schedule) (hh.Schedule, error) { return hh.ScheduleFrom(s) })

	if err != nil {
		log.Errorf("error map schedules")
		return
	}

	var vacancyLatestPublicationTime *hh.VacancyPreview

	for pageNum := 0; ; pageNum++ {

		select {
		case <-ctx.Done():
			log.Infof("analysis canceled for search ID %v", search.ID)
			return
		default:
		}

		params := hh.SearchParameters{
			Text:                   search.SearchText,
			Experience:             hh.Experience(search.Experience),
			Schedules:              schedules,
			DateFrom:               dateFrom,
			AreaID:                 search.RegionID,
			OrderByPublicationTime: true,
			Page:                   pageNum,
			PerPage:                pageSize,
		}
		if err = params.Validate(); err != nil {
			log.Errorf("failed to validate search parameters: %v", err)
			return
		}

		previews, err := v.hhClient.GetVacancies(params)
		if err == nil {
			fetchedTotal += len(previews)
		} else {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeHhApi).Errorf("failed to get vacancies previews: %v", err)
			continue
		}

		if len(previews) == 0 {
			break
		}

		v.analyzeVacanciesByPreviews(ctx, previews, search)

		if vacancyLatestPublicationTime == nil {
			vacancyLatestPublicationTime = &previews[0]
		}
	}

	if vacancyLatestPublicationTime != nil {
		err = v.searches.UpdateLastCheckedVacancy(context.Background(), search.ID, *vacancyLatestPublicationTime)
		if err != nil {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).Errorf("failed to update last checked vacancy: %v", err)
		}
	}

	log.Infof("fetched total %v vacancies for search with id %v", fetchedTotal, search.ID)
}

func (v *VacanciesAnalyzer) analyzeVacanciesByPreviews(ctx context.Context, previews []hh.VacancyPreview,
	search entities.JobSearch) {

	for _, vacancyPreview := range previews {

		select {
		case <-ctx.Done():
			return
		default:
		}

		start := time.Now()
		vacancy, err := v.hhClient.GetVacancy(vacancyPreview.ID)
		metrics.AnalysisStepDuration.WithLabelValues("info_retrieval").Observe(time.Since(start).Seconds())

		if err != nil {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeHhApi).Errorf("failed to get vacancy: %v", err)
		} else if v.analyzeVacancy(ctx, vacancy, search) == nil {
			metrics.HandledVacanciesCounter.Inc()
		}
	}
}

func (v *VacanciesAnalyzer) analyzeVacancy(ctx context.Context, vacancy hh.Vacancy, search entities.JobSearch) error {

	descriptionHash := sha256.Sum256([]byte(vacancy.Description))
	cacheID := strconv.Itoa(search.ID) + hex.EncodeToString(descriptionHash[:])
	if _, found := v.cache.Get(cacheID); found {
		return nil
	}

	start := time.Now()

	matched, err := v.aiService.DoesVacancyMatchSearch(ctx, search, vacancy)
	metrics.AnalysisStepDuration.WithLabelValues("ai_analysis").Observe(time.Since(start).Seconds())

	if err != nil {
		if !errors.Is(err, context.Canceled) {
			log.WithField(logger.ErrorTypeField, logger.ErrorTypeAiApi).
				Errorf("failed to generate response for vacancy %v: %v", vacancy.Url, err)
		}
		return err
	}

	if matched {
		err = v.handleApproveByAI(ctx, vacancy, search)
		if err != nil {
			return err
		}
	} else {
		metrics.RejectedByAiVacanciesCounter.Inc()
	}

	if err = v.cache.Add(cacheID, "", gocache.DefaultExpiration); err != nil {
		log.Errorf("failed to add description to cache: %v", err)
	}

	return nil
}

func (v *VacanciesAnalyzer) handleApproveByAI(ctx context.Context, vacancy hh.Vacancy, search entities.JobSearch) error {

	wasSent, err := v.vacancies.IsSentToUser(ctx, search.UserID, vacancy.ID)
	if err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).
			Errorf("failed to check if vacancy was sent to user: %v", err)
		return err
	}

	if wasSent {
		return nil
	}

	if err = v.vacancies.RecordAsSentToUser(ctx, search.UserID, vacancy.ID); err != nil {
		log.WithField(logger.ErrorTypeField, logger.ErrorTypeDb).
			Errorf("failed to record vacancy as send to user: %v", err)
		return err
	}
	event := events.VacancyFound{Search: search, Name: vacancy.Name, Url: vacancy.Url}
	v.bus.Publish(events.VacancyFoundTopic, event)
	return nil
}

func (v *VacanciesAnalyzer) onSearchDeletedEvent(event events.SearchDeleted) {
	if cancel, ok := v.searchContexts.Load(event.SearchID); ok {
		cancel.(context.CancelFunc)()
		v.searchContexts.Delete(event.SearchID)
	}
}

func tryMapArray[T any, U any](input []T, fn func(T) (U, error)) ([]U, error) {
	result := make([]U, len(input))
	for i, v := range input {
		var err error
		if result[i], err = fn(v); err != nil {
			return nil, err
		}
	}
	return result, nil
}
