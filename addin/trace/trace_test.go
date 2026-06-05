// SPDX-License-Identifier: GPL-2.0-only

package trace

import (
	"log/slog"
	"testing"
	"time"
)

func TestRecordOpAndTailCursor(t *testing.T) {
	b := NewBuffer(0)
	b.RecordOp("a.one", 100*time.Microsecond, true, "", "", "")
	b.RecordOp("a.two", 200*time.Microsecond, false, "a.two: bad", "", "")

	res := b.Tail(0, "", 0)
	if len(res.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(res.Records))
	}
	if res.Records[0].Method != "a.one" || res.Records[0].Level != "info" {
		t.Errorf("rec0 = %+v, want a.one/info", res.Records[0])
	}
	if res.Records[1].Level != "warn" || res.Records[1].Error != "a.two: bad" {
		t.Errorf("rec1 = %+v, want warn + error text", res.Records[1])
	}
	// Cursor: polling with NextSeq returns only newer records.
	if got := b.Tail(res.NextSeq, "", 0); len(got.Records) != 0 {
		t.Errorf("tail after cursor returned %d, want 0", len(got.Records))
	}
	b.RecordOp("a.three", 0, true, "", "", "")
	if got := b.Tail(res.NextSeq, "", 0); len(got.Records) != 1 || got.Records[0].Method != "a.three" {
		t.Errorf("incremental tail = %+v, want only a.three", got.Records)
	}
}

func TestTailLevelFilter(t *testing.T) {
	b := NewBuffer(0)
	b.RecordOp("ok", 0, true, "", "", "")            // info
	b.RecordOp("warn", 0, false, "x", "", "")        // warn
	b.RecordOp("boom", 0, false, "", "panic", "stk") // error
	res := b.Tail(0, "warn", 0)
	if len(res.Records) != 2 {
		t.Fatalf("warn+ filter got %d, want 2", len(res.Records))
	}
	for _, r := range res.Records {
		if r.Level == "info" {
			t.Errorf("info record leaked past warn filter: %+v", r)
		}
	}
}

func TestRingDropsOldestAndCounts(t *testing.T) {
	b := NewBuffer(3)
	for i := 0; i < 5; i++ {
		b.RecordOp("m", 0, true, "", "", "")
	}
	res := b.Tail(0, "", 0)
	if len(res.Records) != 3 {
		t.Fatalf("ring kept %d, want capacity 3", len(res.Records))
	}
	if res.Dropped != 2 {
		t.Errorf("dropped = %d, want 2", res.Dropped)
	}
	if res.Records[0].Seq != 3 {
		t.Errorf("oldest retained Seq = %d, want 3 (1,2 evicted)", res.Records[0].Seq)
	}
}

func TestSlogHandlerCaptured(t *testing.T) {
	b := NewBuffer(0)
	log := slog.New(b.SlogHandler(slog.LevelInfo))
	log.Info("hello", "k", 7)
	log.Debug("ignored") // below the handler minimum
	res := b.Tail(0, "", 0)
	if len(res.Records) != 1 {
		t.Fatalf("captured %d records, want 1 (debug filtered)", len(res.Records))
	}
	r := res.Records[0]
	if r.Method != "" || r.Level != "info" || r.Message == "" {
		t.Errorf("slog record = %+v, want structured info entry", r)
	}
}
