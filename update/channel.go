// SPDX-License-Identifier: GPL-2.0-only

// Package update checks GitHub for a newer Oblikovati release on the running
// build's channel. The host (GUI head) and the CLI both consult it; it owns only
// the version comparison and the GitHub query (behind the [ReleaseSource] seam) —
// callers own how the result is surfaced (a status-bar notice, a window, a line of
// CLI text) and whether to honor the user's "check on startup" preference.
//
// A build's channel is derived from its linker-stamped version
// ([oblikovati.org/build.Version]): a "-nightly" prerelease comes from the rolling
// nightly prerelease, a plain {MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH} from a stable
// release, and "dev" (a local build) from no channel at all — so a developer's build
// never nags.
package update

import "strings"

// Channel is the release stream a build belongs to. The host only ever compares a
// build against the latest release on its OWN channel (a nightly user is offered the
// next nightly, not a far-older stable), mirroring how nightly.yml and release.yml
// publish (see those workflows).
type Channel int

const (
	// Dev is a local build (build.Version "dev" or empty): there is nothing to
	// compare against, so the checker reports a skip and never touches the network.
	Dev Channel = iota
	// Stable is a tagged release from the `release` branch (release.yml).
	Stable
	// Nightly is the rolling prerelease from `develop` (nightly.yml).
	Nightly
)

// nightlySuffix is the semver prerelease identifier cmd/obkversion gives a nightly
// build, followed by ".<timestamp>" (e.g. "0.000200.1.0-nightly.20260614T120000").
const nightlySuffix = "-nightly"

// String renders the channel for user-facing text and logs.
func (c Channel) String() string {
	switch c {
	case Stable:
		return "stable"
	case Nightly:
		return "nightly"
	default:
		return "dev"
	}
}

// DetectChannel classifies a linker-stamped build version into its release channel.
// An empty or "dev" version is a local build (Dev); a "-nightly" prerelease identifier
// (which a timestamp follows) is Nightly; anything else is a Stable release.
func DetectChannel(version string) Channel {
	switch {
	case version == "" || version == "dev":
		return Dev
	case strings.Contains(version, nightlySuffix):
		return Nightly
	default:
		return Stable
	}
}
