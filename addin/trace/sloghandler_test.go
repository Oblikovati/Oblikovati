// SPDX-License-Identifier: GPL-2.0-only

package trace

import (
	"log/slog"
	"strings"
	"testing"
)

// TestSlogHandlerFoldsRecords drives the slog.Handler: level mapping, the Enabled filter,
// and WithAttrs/WithGroup folding attrs (namespaced) onto the message.
func TestSlogHandlerFoldsRecords(t *testing.T) {
	b := NewBuffer(0)
	log := slog.New(b.SlogHandler(slog.LevelDebug))

	log.Debug("tiny")
	log.With("svc", "kernel").WithGroup("op").Info("built", "id", 7)
	log.Warn("careful")
	log.Error("boom", "code", 500)

	res := b.Tail(0, "", 0)
	if len(res.Records) != 4 {
		t.Fatalf("got %d records, want 4", len(res.Records))
	}
	for i, want := range []string{"debug", "info", "warn", "error"} {
		if res.Records[i].Level != want {
			t.Errorf("record %d level = %q, want %q", i, res.Records[i].Level, want)
		}
		if res.Records[i].Method != "" {
			t.Errorf("record %d Method = %q, want empty (a log, not an op)", i, res.Records[i].Method)
		}
	}
	if m := res.Records[1].Message; !strings.Contains(m, "built") || !strings.Contains(m, "svc=kernel") || !strings.Contains(m, "op.id=7") {
		t.Errorf("info message = %q, want it to carry built + svc=kernel + op.id=7", m)
	}
	if m := res.Records[3].Message; !strings.Contains(m, "code=500") {
		t.Errorf("error message = %q, want code=500", m)
	}
}

// TestSlogHandlerEnabledFilter: a record below the handler's level is dropped.
func TestSlogHandlerEnabledFilter(t *testing.T) {
	b := NewBuffer(0)
	h := b.SlogHandler(slog.LevelWarn)
	log := slog.New(h)
	log.Info("dropped")
	log.Warn("kept")
	if res := b.Tail(0, "", 0); len(res.Records) != 1 || res.Records[0].Level != "warn" {
		t.Fatalf("tail = %+v, want only the warn record", res.Records)
	}
}
