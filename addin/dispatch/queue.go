// SPDX-License-Identifier: GPL-2.0-only

// Package dispatch marshals model work from arbitrary goroutines onto a single
// goroutine. The MCP bridge add-in (and any future add-in) calls into the host
// from its own goroutines, but *app.Session is not thread-safe and is owned by the
// head's frame-loop goroutine. Producers Submit a job and block; the frame loop
// Drains the queue once per frame, running each job on its goroutine. This is the
// one concurrency seam — the model is only ever touched from the Drain goroutine.
package dispatch

import (
	"context"
	"errors"
	"sync"
)

// ErrClosed is returned by Submit once the dispatcher has been closed.
var ErrClosed = errors.New("dispatch: dispatcher closed")

// Job is a unit of model work. It runs on the Drain goroutine, so it may safely
// touch the (non-thread-safe) session captured in its closure. It returns the
// serialized result (JSON, by convention) or an error.
type Job func() ([]byte, error)

type result struct {
	out []byte
	err error
}

type request struct {
	job   Job
	reply chan result // buffered (1) so Drain never blocks on an abandoned waiter
}

// Dispatcher is a single-consumer work queue. Construct with New, feed it from any
// goroutine with Submit, and drain it from the owning goroutine with Drain.
type Dispatcher struct {
	queue     chan *request
	done      chan struct{}
	closeOnce sync.Once
	wakeup    func() // optional; see SetWakeup
}

// New returns a dispatcher whose queue holds up to capacity pending jobs before
// Submit blocks for backpressure. capacity < 1 is treated as 1.
func New(capacity int) *Dispatcher {
	if capacity < 1 {
		capacity = 1
	}
	return &Dispatcher{
		queue: make(chan *request, capacity),
		done:  make(chan struct{}),
	}
}

// SetWakeup registers a callback invoked (on the submitting goroutine) immediately after a
// job is enqueued, so a consumer that is asleep waiting for work can be woken to Drain it.
// The head sets this to post an empty window event (#1493): the render-on-demand loop blocks
// when idle, and without a wake an add-in's submitted call would not run until the next OS
// input event. Set it once before any Submit; fn must be safe to call from any goroutine.
func (d *Dispatcher) SetWakeup(fn func()) { d.wakeup = fn }

// Submit enqueues job and blocks until Drain runs it (returning its result), ctx is
// cancelled, or the dispatcher is closed. NOTE: if ctx fires after the job is queued
// but before it runs, Submit returns ctx.Err() yet the job may still run on a later
// Drain — the caller's timeout does not cancel the model mutation.
func (d *Dispatcher) Submit(ctx context.Context, job Job) ([]byte, error) {
	req := &request{job: job, reply: make(chan result, 1)}
	select {
	case d.queue <- req:
		if d.wakeup != nil {
			d.wakeup() // rouse an idle consumer so this job drains promptly (#1493)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-d.done:
		return nil, ErrClosed
	}
	select {
	case r := <-req.reply:
		return r.out, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-d.done:
		return nil, ErrClosed
	}
}

// Drain runs up to max queued jobs on the calling goroutine and returns how many
// ran; max <= 0 drains everything currently queued. Call once per frame.
func (d *Dispatcher) Drain(max int) int {
	ran := 0
	for max <= 0 || ran < max {
		select {
		case req := <-d.queue:
			out, err := req.job()
			req.reply <- result{out: out, err: err}
			ran++
		default:
			return ran
		}
	}
	return ran
}

// Pending reports the number of jobs currently queued (best-effort; for metrics
// and tests).
func (d *Dispatcher) Pending() int { return len(d.queue) }

// Close stops the dispatcher: in-flight and future Submits return ErrClosed.
// Idempotent. Queued-but-undrained jobs are abandoned.
func (d *Dispatcher) Close() { d.closeOnce.Do(func() { close(d.done) }) }
