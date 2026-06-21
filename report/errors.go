// SPDX-License-Identifier: GPL-2.0-only

package report

import "oblikovati.org/crcpost"

// ErrOffline reports that the reporting service could not be reached (DNS, connection, or
// timeout). Callers treat it as a graceful skip — being offline is not a failure to shout
// about. It is the shared [crcpost.ErrOffline] so errors.Is works across the seam.
var ErrOffline = crcpost.ErrOffline
