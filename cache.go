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
	var cb cache.Backend = cacheBackends.NoopBackend{}

	if cache.IsCachingEnabled() {
		gitRootPath, err := git.GetRoot()
		if err != nil {
			return warnFail(cb, err)
		}

		p, err := cacheBackends.GobCachePathXDG(gitRootPath)
		if err != nil {
			return warnFail(cb, err)
		}

		err = os.MkdirAll(filepath.Dir(p), 0o700)
		if err != nil {
			return warnFail(cb, err)
		}

		logger().Debug("cache initialized", "path", p)
		cb = cacheBackends.GobBackend{Path: p}
	}

	return cache.NewCache(cb)
}
