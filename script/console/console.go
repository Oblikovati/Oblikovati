// SPDX-License-Identifier: GPL-2.0-only

// Package console holds the UI-agnostic state of the Script Console (ADR-0028 §8,
// lua-scripting-plan §3) and the Controller that runs a Lua source against the host
// without blocking the caller. The state is mutex-guarded so the engine's print
// callback (which fires on the script worker goroutine) and the UI render thread can
// touch it safely; the head panel only ever reads a Snapshot and calls the Controller.
//
// Splitting the state + run orchestration out of the head keeps it pure Go and
// headless-testable: the cgo ImGui panel becomes a thin renderer over this core.
package console

import (
	"sync"

	"oblikovati/script"
)

// maxOutputLines caps the retained output so a chatty or runaway script (which the
// quota/timeout still stops) cannot grow the buffer without bound. Oldest lines drop.
const maxOutputLines = 5000

// Console is the live state of one Script Console: the captured output, whether a run
// is in flight, and the outcome of the last finished run. It is safe for concurrent
// use — AppendOutput is driven from the script worker goroutine while the UI reads
// Snapshot from the render thread.
type Console struct {
	mu      sync.Mutex
	output  []string
	running bool
	last    script.Result
	hasLast bool
}

// New returns an empty console.
func New() *Console { return &Console{} }

// Snapshot is an immutable view of the console for one UI frame. Output is a copy, so
// the renderer can hold it without racing AppendOutput.
type Snapshot struct {
	Output  []string      // captured print() lines, oldest first
	Running bool          // a script is currently executing
	Last    script.Result // outcome of the most recent finished run
	HasLast bool          // false until the first run completes
}

// AppendOutput records one print() line (no trailing newline), dropping the oldest line
// once the buffer is full. Called from the script worker goroutine via the engine's
// Print callback.
func (c *Console) AppendOutput(line string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output = append(c.output, line)
	if len(c.output) > maxOutputLines {
		c.output = c.output[len(c.output)-maxOutputLines:]
	}
}

// markRunning sets the in-flight flag; the Controller calls it when a run starts.
func (c *Console) markRunning() {
	c.mu.Lock()
	c.running = true
	c.mu.Unlock()
}

// finish records the run outcome and clears the in-flight flag. The Controller calls it
// when a run completes (success, error, or cancellation).
func (c *Console) finish(res script.Result) {
	c.mu.Lock()
	c.running = false
	c.last = res
	c.hasLast = true
	c.mu.Unlock()
}

// Clear empties the output buffer (the console's "clear" action); it leaves the
// last-result and running state untouched.
func (c *Console) Clear() {
	c.mu.Lock()
	c.output = nil
	c.mu.Unlock()
}

// Snapshot returns a consistent copy of the console state for rendering one frame.
func (c *Console) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.output))
	copy(out, c.output)
	return Snapshot{Output: out, Running: c.running, Last: c.last, HasLast: c.hasLast}
}

// Running reports whether a run is currently in flight (a convenience over Snapshot for
// the panel's button-enable checks).
func (c *Console) Running() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}
