package main

import (
	"github.com/sinclairtarget/git-who/internal/cache"
	cacheBackends "github.com/sinclairtarget/git-who/internal/cache/backends"
)

func getCache() cache.Cache {
	var cb cache.Backend = cacheBackends.NoopBackend{}
	if cache.IsCachingEnabled() {
		cb = cacheBackends.GobBackend{Path: "commits.gob"}
	}

	return cache.NewCache(cb)
}
