// SPDX-License-Identifier: GPL-2.0-only

package trace

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"oblikovati/api/wire"
)

// SlogHandler returns a [slog.Handler] that appends every record at or above min into this
// buffer (as a structured entry: Method empty, Message set). Installing it as the default
// logger (slog.SetDefault) means any future kernel logging flows into the same trace the
// router fills, so a single Tail sees both operations and logs interleaved by Seq.
func (b *Buffer) SlogHandler(min slog.Level) slog.Handler {
	return &slogHandler{buf: b, min: min}
}

// slogHandler folds an slog.Record (message + attrs) into a flat [wire.LogRecord]; attrs are
// rendered "key=value" onto the message so the wire shape stays simple.
type slogHandler struct {
	buf    *Buffer
	min    slog.Level
	prefix string // accumulated "key=value" attrs from WithAttrs, space-separated
	group  string // current group name (prefixes attr keys)
}

func (h *slogHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.min }

// Handle renders the record and pushes it to the buffer (never erroring — a dropped log must
// not break the operation that logged it).
func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	parts := make([]string, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		parts = append(parts, h.render(a))
		return true
	})
	h.buf.push(wire.LogRecord{
		Level:      slogLevelName(r.Level),
		Message:    joinMessage(r.Message, h.prefix, strings.Join(parts, " ")),
		TimeMillis: r.Time.UnixMilli(),
	})
	return nil
}

// WithAttrs returns a handler that prepends the given attrs to every record's message.
func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	rendered := make([]string, len(attrs))
	for i, a := range attrs {
		rendered[i] = h.render(a)
	}
	return &slogHandler{buf: h.buf, min: h.min, group: h.group, prefix: joinFields(h.prefix, strings.Join(rendered, " "))}
}

// WithGroup returns a handler whose subsequent attr keys are namespaced under name.
func (h *slogHandler) WithGroup(name string) slog.Handler {
	return &slogHandler{buf: h.buf, min: h.min, prefix: h.prefix, group: joinGroup(h.group, name)}
}

// render formats one attr as "key=value", namespaced by the current group.
func (h *slogHandler) render(a slog.Attr) string {
	key := a.Key
	if h.group != "" {
		key = h.group + "." + key
	}
	return fmt.Sprintf("%s=%v", key, a.Value.Any())
}

// joinMessage assembles the final message text from the base message and any attr fields.
func joinMessage(msg, prefix, attrs string) string {
	return joinFields(msg, joinFields(prefix, attrs))
}

func joinFields(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + " " + b
	}
}

func joinGroup(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}

// slogLevelName maps an slog.Level to the trace's lowercase level vocabulary.
func slogLevelName(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "error"
	case l >= slog.LevelWarn:
		return "warn"
	case l >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}
