package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sinclairtarget/git-who/internal/cache"
	cacheBackends "github.com/sinclairtarget/git-who/internal/cache/backends"
	"github.com/sinclairtarget/git-who/internal/git"
)

func warnFail(cb cache.Backend, err error) cache.Cache {
	logger().Warn(
		fmt.Sprintf("failed to initialize cache: %v", err),
	)
	logger().Warn("disabling caching")
	return cache.NewCache(cb)
}

// getCache returns the repository's cache.
func getCache(
	ctx context.Context,
	gitRootPath string,
	repoFiles git.RepoConfigFiles,
) cache.Cache {
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

	dirname := cacheBackends.GobCacheDir(cacheStorageDir, gitRootPath)
	err = os.MkdirAll(dirname, 0o700)
	if err != nil {
		return warnFail(fallback, err)
	}

	filename, err := cacheBackends.GobCacheFilename(ctx, repoFiles)
	if err != nil {
		return warnFail(fallback, err)
	}

	p := filepath.Join(dirname, filename)
	logger().Debug("cache initialized", "path", p)
	return cache.NewCache(&cacheBackends.GobBackend{Path: p, Dir: dirname})
}
