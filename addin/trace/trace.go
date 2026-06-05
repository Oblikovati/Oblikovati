// SPDX-License-Identifier: GPL-2.0-only

// Package trace is the host's in-memory operation trace: a thread-safe ring buffer of
// [wire.LogRecord]s that the add-in router fills (one record per served method call, with
// timing and outcome) and that any [log/slog] logger can append to. It backs the diagnostics
// surface (wire.MethodLogsTail) so a driver can watch the kernel work in real time, see
// detailed errors, and catch panics while stress-testing — there is no other logging in the
// core today, so this trace is the single source of truth for "what the kernel just did".
package trace

import (
	"sync"
	"time"

	"github.com/Oblikovati/api/wire"
)

// DefaultCapacity is the ring size when [NewBuffer] is given a non-positive capacity — large
// enough to survive a burst of stress-test calls between polls without dropping.
const DefaultCapacity = 4096

// defaultTailMax caps a Tail with no explicit Max, so a poll can't return an unbounded slice.
const defaultTailMax = 500

// Buffer is a bounded, monotonic record stream. Records carry a gap-free Seq (assigned on
// push); when the ring overflows the oldest are dropped and counted, so a slow poller learns
// it missed some. All methods are safe for concurrent use.
type Buffer struct {
	mu       sync.Mutex
	records  []wire.LogRecord
	capacity int
	nextSeq  uint64
	dropped  uint64
}

// NewBuffer returns a trace buffer holding up to capacity records (DefaultCapacity if <= 0).
func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Buffer{capacity: capacity}
}

// push assigns the next Seq, stamps the time if unset, appends, and evicts the oldest beyond
// capacity (counting the eviction as dropped).
func (b *Buffer) push(r wire.LogRecord) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextSeq++
	r.Seq = b.nextSeq
	if r.TimeMillis == 0 {
		r.TimeMillis = time.Now().UnixMilli()
	}
	b.records = append(b.records, r)
	if over := len(b.records) - b.capacity; over > 0 {
		b.dropped += uint64(over)
		b.records = b.records[over:]
	}
}

// RecordOp appends an operation entry: a router method call with its duration and outcome.
// A non-empty panicMsg marks a recovered kernel panic (the bug signal) and carries its stack.
func (b *Buffer) RecordOp(method string, dur time.Duration, ok bool, errMsg, panicMsg, stack string) {
	b.push(wire.LogRecord{
		Level:          opLevel(ok, panicMsg),
		Method:         method,
		DurationMicros: dur.Microseconds(),
		OK:             ok,
		Error:          errMsg,
		Panic:          panicMsg,
		Stack:          stack,
	})
}

// opLevel maps an operation outcome to a severity: panic ⇒ error, plain failure ⇒ warn, ok ⇒ info.
func opLevel(ok bool, panicMsg string) string {
	switch {
	case panicMsg != "":
		return "error"
	case !ok:
		return "warn"
	default:
		return "info"
	}
}

// Tail returns records with Seq > sinceSeq, at or above the given minimum level (empty ⇒ all),
// oldest-first and capped to max (defaultTailMax if <= 0). NextSeq is the cursor to poll with
// next (the highest Seq scanned this call), so repeated tailing yields only new records.
func (b *Buffer) Tail(sinceSeq uint64, level string, max int) wire.LogsResult {
	if max <= 0 {
		max = defaultTailMax
	}
	minRank := levelRank(level)
	b.mu.Lock()
	defer b.mu.Unlock()
	res := wire.LogsResult{NextSeq: sinceSeq, Dropped: b.dropped}
	for _, r := range b.records {
		if r.Seq <= sinceSeq {
			continue
		}
		res.NextSeq = r.Seq
		if levelRank(r.Level) >= minRank {
			res.Records = append(res.Records, r)
		}
		if len(res.Records) >= max {
			break
		}
	}
	return res
}

// levelRank orders severities so a minimum-level filter is a simple comparison; an unknown or
// empty level ranks below everything (so the "all" filter and unlabeled records always pass).
func levelRank(level string) int {
	switch level {
	case "debug":
		return 1
	case "info":
		return 2
	case "warn":
		return 3
	case "error":
		return 4
	default:
		return 0
	}
}
