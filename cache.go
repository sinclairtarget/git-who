package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sinclairtarget/git-who/internal/cache"
	cacheBackends "github.com/sinclairtarget/git-who/internal/cache/backends"
	"github.com/sinclairtarget/git-who/internal/git"
)

func warnFail(cb cache.Backend, err error) cache.Cache {
	logger().Warn(
		fmt.Sprintf("failed to create initialize cache: %v", err),
	)
	logger().Warn("disabling caching")
	return cache.NewCache(cb)
}

func getCache() cache.Cache {
	var fallback cache.Backend = cacheBackends.NoopBackend{}

	if !cache.IsCachingEnabled() {
		return cache.NewCache(fallback)
	}

	cacheStorageDir, err := cache.CacheStorageDir(
		cacheBackends.GobBackendName,
	)
	if err != nil {
		return warnFail(fallback, err)
	}

	gitRootPath, err := git.GetRoot()
	if err != nil {
		return warnFail(fallback, err)
	}

	p := cacheBackends.GobCachePath(cacheStorageDir, gitRootPath)
	err = os.MkdirAll(filepath.Dir(p), 0o700)
	if err != nil {
		return warnFail(fallback, err)
	}

	logger().Debug("cache initialized", "path", p)
	return cache.NewCache(&cacheBackends.GobBackend{Path: p})
}
