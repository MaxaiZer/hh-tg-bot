package services

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/maxaizer/hh-parser/internal/entities"
	gocache "github.com/patrickmn/go-cache"
	"strconv"
)

type cacheHelper struct {
	cache *gocache.Cache
}

func newCacheHelper(cache *gocache.Cache) *cacheHelper {
	return &cacheHelper{
		cache: cache,
	}
}

func (h *cacheHelper) cacheForSearch(searchID int, vacancy entities.Vacancy) {
	cacheID := createVacancyCacheID(searchID, vacancy.Description)
	h.cache.Set(cacheID, "", gocache.DefaultExpiration)
}

func (h *cacheHelper) isCachedForSearch(searchID int, vacancy entities.Vacancy) bool {
	_, found := h.cache.Get(createVacancyCacheID(searchID, vacancy.Description))
	return found
}

func createVacancyCacheID(searchID int, description string) string {
	descriptionHash := sha256.Sum256([]byte(description))
	return strconv.Itoa(searchID) + hex.EncodeToString(descriptionHash[:])
}
