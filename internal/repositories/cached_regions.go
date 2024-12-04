package repositories

import (
	"context"
	gocache "github.com/patrickmn/go-cache"
	"time"
)

type regionRepository interface {
	GetIdByName(ctx context.Context, name string) (string, error)
}

type CachedRegions struct {
	repo  regionRepository
	cache *gocache.Cache
}

func NewCachedRegions(repo regionRepository) *CachedRegions {
	return &CachedRegions{repo: repo, cache: gocache.New(10*time.Minute, 20*time.Minute)}
}

func (c CachedRegions) GetIdByName(ctx context.Context, name string) (string, error) {
	if value, found := c.cache.Get(name); found {
		return value.(string), nil
	}

	id, err := c.repo.GetIdByName(ctx, name)
	if id != "" {
		c.cache.Set(name, id, gocache.DefaultExpiration)
	}

	return id, err
}
