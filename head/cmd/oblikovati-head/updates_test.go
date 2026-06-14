//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/update"
)

// upToDate / available are canned results for the service-gating tests.
func upToDate() update.Result {
	return update.Result{Channel: update.Stable, Current: "0.0.1"}
}
func available() update.Result {
	return update.Result{
		Channel: update.Stable, Current: "0.0.1", UpdateAvailable: true,
		Latest: update.Release{Version: "0.0.2", HTMLURL: "https://gh/r"},
	}
}

func TestServiceUpdatesManualAlwaysShows(t *testing.T) {
	s := app.NewSession()
	p := &updatePoller{}
	p.outcome.Store(&updateOutcome{res: upToDate(), manual: true})
	serviceUpdates(s, p)
	if s.PendingUpdate() == nil {
		t.Fatal("a manual check must open the window even when up to date")
	}
}

func TestServiceUpdatesAutoSuppressesUpToDate(t *testing.T) {
	s := app.NewSession()
	p := &updatePoller{}
	p.outcome.Store(&updateOutcome{res: upToDate(), manual: false})
	serviceUpdates(s, p)
	if s.PendingUpdate() != nil {
		t.Fatal("the silent startup check must not nag when up to date")
	}
}

func TestServiceUpdatesAutoShowsAvailable(t *testing.T) {
	s := app.NewSession()
	p := &updatePoller{}
	p.outcome.Store(&updateOutcome{res: available(), manual: false})
	serviceUpdates(s, p)
	if s.PendingUpdate() == nil || !s.PendingUpdate().UpdateAvailable {
		t.Fatal("the startup check must surface a real update")
	}
}

func TestServiceUpdatesNoOutcomeIsNoop(t *testing.T) {
	s := app.NewSession()
	serviceUpdates(s, &updatePoller{})
	if s.PendingUpdate() != nil {
		t.Fatal("no completed check should leave the window closed")
	}
}
