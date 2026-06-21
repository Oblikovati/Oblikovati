// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/app/options"

// Anonymous usage-statistics surface (#1182). The session owns the user-visible opt-out
// preference; the head reads it before submitting an installation snapshot during the
// startup update-check (the network I/O lives in the head, off the UI thread, exactly like
// the update check — see usagestats and head/cmd/oblikovati-head/metrics.go).

// TelemetryEnabled reports whether anonymous usage statistics may be shared (the persisted
// Telemetry.ShareUsageStatistics preference, on by default — it is opt-out).
func (s *Session) TelemetryEnabled() bool { return s.appOptions.Telemetry.ShareUsageStatistics }

// SetTelemetryEnabled stores the usage-statistics opt-out preference, persisting it to the
// user's options file.
func (s *Session) SetTelemetryEnabled(on bool) error {
	s.appOptions.Telemetry = options.Telemetry{ShareUsageStatistics: on}
	return s.saveOptions()
}
