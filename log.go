package main

import (
	"log/slog"
)

var pkgLogger *slog.Logger

func logger() *slog.Logger {
	if pkgLogger == nil {
		pkgLogger = slog.Default().With("package", "main")
	}

	return pkgLogger
}
