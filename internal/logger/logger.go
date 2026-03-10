package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

type Handler struct {
	mu      sync.Mutex
	out     io.Writer
	colored bool
	attrs   []slog.Attr
}

func newHandler(out io.Writer) *Handler {
	isTerminal := false
	if f, ok := out.(*os.File); ok {
		isTerminal = isatty(f)
	}
	return &Handler{out: out, colored: isTerminal}
}

func isatty(f *os.File) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func (h *Handler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(clone.attrs, attrs...)
	return &clone
}

func (h *Handler) WithGroup(_ string) slog.Handler {
	return h
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var sb strings.Builder

	timestamp := r.Time.Format("2006-01-02 15:04:05")
	if h.colored {
		sb.WriteString(colorGray + timestamp + colorReset)
	} else {
		sb.WriteString(timestamp)
	}

	sb.WriteString("  ")

	levelStr := levelText(r.Level)
	if h.colored {
		sb.WriteString(levelColor(r.Level) + fmt.Sprintf("%-7s", levelStr) + colorReset)
	} else {
		sb.WriteString(fmt.Sprintf("%-7s", levelStr))
	}

	sb.WriteString("  ")

	msg := fmt.Sprintf("%-30s", r.Message)
	if h.colored {
		sb.WriteString(colorBold + msg + colorReset)
	} else {
		sb.WriteString(msg)
	}

	for _, a := range h.attrs {
		sb.WriteString("  ")
		sb.WriteString(formatAttr(a, h.colored))
	}

	r.Attrs(func(a slog.Attr) bool {
		sb.WriteString("  ")
		sb.WriteString(formatAttr(a, h.colored))
		return true
	})

	sb.WriteString("\n")

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := fmt.Fprint(h.out, sb.String())
	return err
}

func formatAttr(a slog.Attr, colored bool) string {
	key := a.Key
	val := a.Value.String()
	if colored {
		return colorCyan + key + colorReset + "=" + val
	}
	return key + "=" + val
}

func levelText(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

func levelColor(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return colorRed
	case l >= slog.LevelWarn:
		return colorYellow
	case l >= slog.LevelInfo:
		return colorGreen
	default:
		return colorGray
	}
}

var Default *slog.Logger

func init() {
	handler := newHandler(os.Stdout)
	Default = slog.New(handler)
	slog.SetDefault(Default)
}

func Info(msg string, args ...any) {
	Default.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Default.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Default.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	Default.Error(msg, args...)
	time.Sleep(100 * time.Millisecond)
	os.Exit(1)
}
