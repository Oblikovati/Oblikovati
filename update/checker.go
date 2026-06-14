// SPDX-License-Identifier: GPL-2.0-only

package update

import (
	"context"
	"errors"
)

// Checker compares the running build against the latest release on its channel through
// an injected [ReleaseSource]. Construct it once and reuse it for both the startup
// auto-check and the manual "Check for Updates" command.
type Checker struct{ source ReleaseSource }

// NewChecker returns a checker that queries source.
func NewChecker(source ReleaseSource) *Checker { return &Checker{source: source} }

// Check reports whether a newer release than current exists on current's channel.
//
// It never returns an error for the expected "nothing to offer" cases — a dev build, an
// offline machine, or a channel with no release — those come back as Skipped with a
// reason, so the GUI can show "you're up to date / offline" and the CLI can stay quiet.
// Only an unexpected failure (bad response, HTTP 5xx) is returned as an error.
//
//	res, err := chk.Check(ctx, build.Version)
//	if err == nil && res.UpdateAvailable { notify(res.Latest.HTMLURL) }
func (c *Checker) Check(ctx context.Context, current string) (Result, error) {
	res := Result{Channel: DetectChannel(current), Current: current}
	if res.Channel == Dev {
		return skip(res, "development build"), nil
	}
	latest, err := c.source.Latest(ctx, res.Channel)
	if errors.Is(err, ErrOffline) {
		return skip(res, "offline"), nil
	}
	if errors.Is(err, ErrNoRelease) {
		return skip(res, "no published release"), nil
	}
	if err != nil {
		return res, err
	}
	res.Latest = latest
	res.UpdateAvailable = IsNewer(latest.Version, current)
	return res, nil
}

// skip marks a result as a graceful no-op with the given reason.
func skip(res Result, reason string) Result {
	res.Skipped = true
	res.SkipReason = reason
	return res
}
