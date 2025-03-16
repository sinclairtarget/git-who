package backends

import (
	"log/slog"
)

var pkgLogger *slog.Logger

func logger() *slog.Logger {
	if pkgLogger == nil {
		pkgLogger = slog.Default().With("package", "cache.backends")
	}

	return pkgLogger
}
