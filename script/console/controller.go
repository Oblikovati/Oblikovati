// SPDX-License-Identifier: GPL-2.0-only

package console

import (
	"context"
	"sync"

	"oblikovati.org/script"
	"oblikovati.org/script/runner"
)

// Controller drives one Script Console: it starts a Lua source on a background
// goroutine (so the UI render thread never blocks), streams print() output into the
// Console, records the outcome, and lets the user cancel an in-flight run. It owns the
// run's context so Stop unwinds a looping or long script promptly (ADR-0028 §5, §7).
type Controller struct {
	runner  *runner.ScriptRunner
	console *Console
	limits  script.Limits

	mu     sync.Mutex
	cancel context.CancelFunc // non-nil only while a run is in flight
}

// NewController wires a runner and the resource limits behind a fresh Console. Use
// runner.DefaultGUILimits for the head so a buggy console script is bounded.
func NewController(run *runner.ScriptRunner, limits script.Limits) *Controller {
	return &Controller{runner: run, console: New(), limits: limits}
}

// Console exposes the live state the UI renders (its Snapshot/Running/Clear).
func (c *Controller) Console() *Console { return c.console }

// Run starts source on a background goroutine and returns immediately. It returns
// runner.ErrBusy if a run is already in flight (the "Run" button must stay disabled
// while running). Output streams to the Console as the script prints; the outcome is
// recorded when it finishes.
func (c *Controller) Run(source string) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.mu.Unlock()
		return runner.ErrBusy
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.mu.Unlock()

	c.console.markRunning()
	go c.execute(ctx, source)
	return nil
}

// execute runs the script to completion off the UI thread, recording the result (and
// surfacing a runner-level error, e.g. ErrBusy, as the Result.Err) before clearing the
// in-flight context.
func (c *Controller) execute(ctx context.Context, source string) {
	res, err := c.runner.Run(ctx, source, c.limits, c.console.AppendOutput)
	if err != nil {
		res = script.Result{Err: err}
	}
	c.console.finish(res)
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel() // release the context's resources
		c.cancel = nil
	}
	c.mu.Unlock()
}

// Stop cancels an in-flight run (the console "Stop" button). It is a no-op when idle.
func (c *Controller) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
