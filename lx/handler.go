package lx

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func ParseLogLevel(level string) slog.Leveler {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Default() *slog.Logger {
	return slog.New(Handle(os.Stderr, Options{}))
}

func New(w io.Writer, opts Options) *slog.Logger {
	return slog.New(Handle(w, opts))
}

func Handle(w io.Writer, opts Options) slog.Handler {
	opts.Level = cmp.Or[slog.Leveler](opts.Level, slog.LevelDebug)
	opts.TimeFormat = cmp.Or(opts.TimeFormat, "01-02 15:04:05")
	return &handler{opts: opts, w: cmp.Or[io.Writer](w, os.Stderr)}
}

type Options struct {
	AddSource  bool
	Level      slog.Leveler
	TimeFormat string
}

var _ slog.Handler = (*handler)(nil)

type handler struct {
	opts Options
	w    io.Writer

	attrs []slog.Attr
	group string

	mu sync.Mutex
}

// WithAttrs implements [slog.Handler].
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs, attrs...)
	return h
}

// WithGroup implements [slog.Handler].
func (h *handler) WithGroup(name string) slog.Handler {
	h.group = name
	return h
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	var buf bytes.Buffer

	// write time
	if !r.Time.IsZero() {
		fmt.Fprintf(&buf, "[%s] ", r.Time.Format(h.opts.TimeFormat))
	}

	fmt.Fprintf(&buf, "%s ", r.Level.String())

	// write source
	if h.opts.AddSource {
		if f, _ := runtime.CallersFrames([]uintptr{r.PC}).Next(); f.File != "" {
			fmt.Fprintf(&buf, "%s:%d ", filepath.Base(f.File), f.Line)
		}
	}

	for attr := range r.Attrs {
		if attr.Key == "tag" {
			fmt.Fprintf(&buf, "[%s] ", attr.Value.String())
			break
		}
	}

	// write message
	buf.WriteString(r.Message)
	buf.WriteByte(' ')

	// write attributes
	for attr := range r.Attrs {
		if attr.Key != "tag" {
			h.writeAttr(&buf, attr)
		}
	}

	buf.Truncate(buf.Len() - 1)
	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *handler) writeAttr(w io.Writer, attr slog.Attr) {
	fmt.Fprintf(w, "%s=%q ", attr.Key, attr.Value.String())
}
