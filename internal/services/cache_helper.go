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

	var vacancy hh.Vacancy
	var err error

	if cached, found := h.cache.Get(ID); found {
		vacancy = cached.(hh.Vacancy)
	} else {
		start := time.Now()
		vacancy, err = h.hhClient.GetVacancy(ID)
		metrics.AnalysisStepDuration.WithLabelValues("info_retrieval").Observe(time.Since(start).Seconds())

		h.cache.Set(ID, vacancy, gocache.DefaultExpiration)
	}

	if err != nil {
		return nil, err
	}
	return &vacancy, nil
}

func (h *cacheHelper) cacheByDescription(searchID int, description string) {
	cacheID := createVacancyCacheID(searchID, description)
	h.cache.Set(cacheID, "", gocache.DefaultExpiration)
}

func (h *cacheHelper) isInCacheByDescription(searchID int, description string) bool {
	_, found := h.cache.Get(createVacancyCacheID(searchID, description))
	return found
}

func createVacancyCacheID(searchID int, description string) string {
	descriptionHash := sha256.Sum256([]byte(description))
	return strconv.Itoa(searchID) + hex.EncodeToString(descriptionHash[:])
}
