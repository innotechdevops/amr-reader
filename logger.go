package amrreader

import (
	"log"
	"log/slog"
)

type Logger struct {
	Enabled bool
}

func (l Logger) Info(msg string, args ...any) {
	if l.Enabled {
		slog.Info(msg, args...)
	}
}

func (l Logger) Error(msg string, args ...any) {
	if l.Enabled {
		slog.Error(msg, args...)
	}
}

func (l Logger) Fatal(v ...any) {
	if l.Enabled {
		log.Fatal(v...)
	}
}
