package services

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/metrics"
	gocache "github.com/patrickmn/go-cache"
	"strconv"
	"time"
)

type cacheHelper struct {
	hhClient *hh.Client
	cache    *gocache.Cache
}

func newCacheHelper(client *hh.Client, cache *gocache.Cache) *cacheHelper {
	return &cacheHelper{
		hhClient: client,
		cache:    cache,
	}
}

func (h *cacheHelper) getVacancyByID(ID string) (*hh.Vacancy, error) {

	if cached, found := h.cache.Get(ID); found {
		vacancy := cached.(hh.Vacancy)
		return &vacancy, nil
	}

	start := time.Now()
	vacancy, err := h.hhClient.GetVacancy(ID)
	if err != nil {
		return nil, err
	}

	metrics.AnalysisStepDuration.WithLabelValues("info_retrieval").Observe(time.Since(start).Seconds())
	h.cache.Set(ID, vacancy, gocache.DefaultExpiration)
	return &vacancy, nil
}

func (h *cacheHelper) cacheForSearch(searchID int, vacancy hh.Vacancy) {
	cacheID := createVacancyCacheID(searchID, vacancy.Description)
	h.cache.Set(cacheID, "", gocache.DefaultExpiration)
}

func (h *cacheHelper) isCachedForSearch(searchID int, vacancy hh.Vacancy) bool {
	_, found := h.cache.Get(createVacancyCacheID(searchID, vacancy.Description))
	return found
}

func createVacancyCacheID(searchID int, description string) string {
	descriptionHash := sha256.Sum256([]byte(description))
	return strconv.Itoa(searchID) + hex.EncodeToString(descriptionHash[:])
}
