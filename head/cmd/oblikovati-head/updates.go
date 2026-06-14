//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"oblikovati.org/app"
	"oblikovati.org/build"
	"oblikovati.org/update"
)

// updateHTTPTimeout bounds the GitHub query so a slow or missing network never blocks the
// frame loop (the check runs off-thread, but the timeout also frees that goroutine).
const updateHTTPTimeout = 8 * time.Second

// updateOutcome pairs a check result with whether it came from a manual Help ▸ Check for
// Updates: a manual check always opens the window (even "you're up to date"), while the
// silent startup check only surfaces when there is actually a newer release to offer.
type updateOutcome struct {
	res    update.Result
	manual bool
}

// updatePoller runs the GitHub update check off the UI thread and hands the result back
// to the (single-threaded) session through an atomic pointer the frame loop drains — the
// session is never touched from the network goroutine.
type updatePoller struct {
	checker *update.Checker
	version string
	outcome atomic.Pointer[updateOutcome]
	busy    atomic.Bool // one in-flight check at a time
}

// newUpdatePoller builds a poller querying the published GitHub repository.
func newUpdatePoller() *updatePoller {
	src := update.NewGitHubSource(update.DefaultOwner, update.DefaultRepo, &http.Client{Timeout: updateHTTPTimeout})
	return &updatePoller{checker: update.NewChecker(src), version: build.Version}
}

// launch starts a check unless one is already running. manual marks a user-triggered
// check (vs. the startup auto-check) so serviceUpdates knows whether to pop the window.
func (p *updatePoller) launch(manual bool) {
	if !p.busy.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer p.busy.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), updateHTTPTimeout)
		defer cancel()
		res, err := p.checker.Check(ctx, p.version)
		if err != nil {
			res = update.Result{
				Channel: update.DetectChannel(p.version), Current: p.version,
				Skipped: true, SkipReason: "check failed",
			}
		}
		p.outcome.Store(&updateOutcome{res: res, manual: manual})
	}()
}

// take returns and clears a completed check's outcome, if any.
func (p *updatePoller) take() (*updateOutcome, bool) {
	o := p.outcome.Swap(nil)
	return o, o != nil
}

// startupUpdateCheck launches the silent auto-check when the user has it enabled.
func startupUpdateCheck(s *app.Session, p *updatePoller) {
	if s.UpdateChecksEnabled() {
		p.launch(false)
	}
}

// serviceUpdates is called each frame: it starts a manual check when Help ▸ Check for
// Updates was clicked, and publishes any completed outcome into the session — always for
// a manual check, but only on a real update for the silent startup check.
func serviceUpdates(s *app.Session, p *updatePoller) {
	if s.TakeUpdateCheckRequest() {
		p.launch(true)
	}
	o, ok := p.take()
	if !ok {
		return
	}
	if o.manual || o.res.UpdateAvailable {
		s.ShowUpdateResult(o.res)
	}
}
