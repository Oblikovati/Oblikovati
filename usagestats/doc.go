// SPDX-License-Identifier: GPL-2.0-only

// Package usagestats assembles and submits an anonymous installation [Snapshot] — the
// machine's OS/architecture, hardware (RAM, CPU, storage, GPU), Vulkan/MoltenVK version,
// Oblikovati version, and installed add-ins — to the stats.oblikovati.org endpoint during
// the startup update-check (opt-out; see the head's telemetry preference).
//
// It mirrors the sibling report and update packages: the network I/O lives behind a thin
// [httpDoer] seam so the flow is unit-testable without a live server, connectivity failures
// map to [ErrOffline] so a missing network is a graceful skip, and the open POST carries an
// Authorization token equal to the CRC-32 (IEEE) of the exact JSON body (see [Token]).
//
// The snapshot is anonymous: [MachineUUID] is a random identifier generated on the machine's
// first run and stored in the user's globals file — it is not derived from any hardware or
// user identity, and exists only so the service keeps one current row per machine instead of
// inflating the install count on every report.
package usagestats
