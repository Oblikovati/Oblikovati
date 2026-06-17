// SPDX-License-Identifier: GPL-2.0-only

package report

import "errors"

// ErrOffline reports that the reporting service could not be reached (DNS, connection, or
// timeout). Callers treat it as a graceful skip — being offline is not a failure to shout
// about — mirroring update.ErrOffline.
var ErrOffline = errors.New("report: reporting server unreachable")
