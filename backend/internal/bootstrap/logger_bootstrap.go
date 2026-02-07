package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"

	"github.com/getarcaneapp/arcane/backend/internal/config"
)

// timeFilterHandler wraps a slog.Handler and removes redundant time attributes
// from grouped attributes (like request.time and response.time from slog-gin)
type timeFilterHandler struct {
	handler slog.Handler
}

func (h *timeFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *timeFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	// Filter out time attributes from groups
	var filteredAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		if a.Value.Kind() == slog.KindGroup {
			filtered := filterGroupTimeAttrs(a)
			filteredAttrs = append(filteredAttrs, filtered)
		} else {
			filteredAttrs = append(filteredAttrs, a)
		}
		return true
	})

	// Create a new record without the original attrs
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newRecord.AddAttrs(filteredAttrs...)

	return h.handler.Handle(ctx, newRecord)
}

func filterGroupTimeAttrs(a slog.Attr) slog.Attr {
	if a.Value.Kind() != slog.KindGroup {
		return a
	}

	var filtered []slog.Attr
	for _, attr := range a.Value.Group() {
		// Skip "time" attributes within groups (request.time, response.time)
		if attr.Key == "time" {
			continue
		}
		// Recursively filter nested groups
		if attr.Value.Kind() == slog.KindGroup {
			filtered = append(filtered, filterGroupTimeAttrs(attr))
		} else {
			filtered = append(filtered, attr)
		}
	}

	return slog.Group(a.Key, anySlice(filtered)...)
}

func anySlice(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, a := range attrs {
		result[i] = a
	}
	return result
}

func (h *timeFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &timeFilterHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *timeFilterHandler) WithGroup(name string) slog.Handler {
	return &timeFilterHandler{handler: h.handler.WithGroup(name)}
}

func SetupGinLogger(cfg *config.Config) {
	var lvl slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	lv := new(slog.LevelVar)
	lv.Set(lvl)

	var h slog.Handler
	if cfg.LogJson {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	} else {
		h = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      lv,
			TimeFormat: "Jan 02 15:04:05.000",
		})
	}

	// Wrap with timeFilterHandler to remove redundant time attributes from slog-gin
	h = &timeFilterHandler{handler: h}

	slog.SetDefault(slog.New(h))
}
