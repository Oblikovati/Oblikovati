// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import "errors"

// ErrOffline reports that the telemetry service could not be reached (DNS, connection, or
// timeout). Callers treat it as a graceful skip — being offline is not a failure to shout
// about — mirroring report.ErrOffline and update.ErrOffline.
var ErrOffline = errors.New("usagestats: telemetry server unreachable")
