// SPDX-License-Identifier: GPL-2.0-only

package usagestats

// DefaultEndpoint is the public telemetry service that ingests a [Snapshot]. It is
// overridable (NewSubmitter takes the URL) so a local instance can be targeted during
// development and tests.
const DefaultEndpoint = "https://stats.oblikovati.org/report"
