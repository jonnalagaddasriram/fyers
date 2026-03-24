package infra_test

import (
	"io"
	"log/slog"
)

// noopLogger returns a logger that discards all output — used in tests so
// Replicator log output doesn't pollute test stdout.
func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
