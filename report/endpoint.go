// SPDX-License-Identifier: GPL-2.0-only

package report

// DefaultEndpoint is the public reporting service that ingests a [Payload], queues it, and
// opens a GitHub issue. It is overridable (NewSubmitter takes the URL) so a local instance
// can be targeted during development and tests.
const DefaultEndpoint = "https://reporting.oblikovati.org/report"
