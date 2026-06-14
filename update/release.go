// SPDX-License-Identifier: GPL-2.0-only

package update

import (
	"context"
	"errors"
)

// Release is a published release on a channel: the version to compare against and the
// URL of its GitHub page (where the user downloads it).
type Release struct {
	Version string
	HTMLURL string
	Channel Channel
}

// Result is the outcome of a check, designed so a caller never has to interpret an
// error to render the common cases: an offline machine, a dev build, or a channel with
// no published release all come back as a graceful Skipped result, not an error. Only an
// unexpected failure (a malformed response, an HTTP 5xx) is returned as an error.
type Result struct {
	Channel         Channel
	Current         string  // the running build's version
	Latest          Release // the newest release on the channel; zero when Skipped
	UpdateAvailable bool
	Skipped         bool
	SkipReason      string // human-readable: "development build", "offline", "no published release"
}

// ErrOffline marks a transport-level failure (no network, DNS failure, timeout): the
// checker converts it into a Skipped result so the auto-check stays silent offline, per
// the requirement to skip the verification when there is no internet.
var ErrOffline = errors.New("update: release server unreachable")

// ErrNoRelease marks a channel that has no published release yet (GitHub 404): also a
// graceful skip, not a failure the user should see.
var ErrNoRelease = errors.New("update: no release published on channel")

// ReleaseSource fetches the latest release on a channel. It is the seam tests fake and
// [GitHubSource] implements, keeping the network I/O out of the comparison logic.
type ReleaseSource interface {
	Latest(ctx context.Context, channel Channel) (Release, error)
}
