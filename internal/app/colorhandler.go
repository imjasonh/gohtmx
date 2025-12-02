package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

type ColorHandler struct {
	w     io.Writer
	level slog.Level
	mu    sync.Mutex
}

func NewColorHandler(w io.Writer, level slog.Level) *ColorHandler {
	return &ColorHandler{w: w, level: level}
}

func (h *ColorHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.level }

// Simple implementation - for a production handler you'd want to handle groups
func (h *ColorHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *ColorHandler) WithGroup(string) slog.Handler      { return h }

func (h *ColorHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Time in gray
	fmt.Fprintf(h.w, "%s%s%s ", colorGray, r.Time.Format("15:04:05"), colorReset)

	// Level with color
	levelStr := r.Level.String()
	levelColor := ""
	switch r.Level {
	case slog.LevelError:
		levelColor = colorRed
	case slog.LevelWarn:
		levelColor = colorYellow
	case slog.LevelInfo:
		levelColor = colorBlue
	case slog.LevelDebug:
		levelColor = colorGray
	}
	fmt.Fprintf(h.w, "%s%-5s%s ", levelColor, levelStr, colorReset)

	// Message
	fmt.Fprint(h.w, r.Message)

	// Attributes in gray
	if r.NumAttrs() > 0 {
		fmt.Fprintf(h.w, " %s", colorGray)
		r.Attrs(func(a slog.Attr) bool {
			fmt.Fprintf(h.w, " %s=%v", a.Key, a.Value)
			return true
		})
		fmt.Fprint(h.w, colorReset)
	}

	fmt.Fprintln(h.w)
	return nil
}
