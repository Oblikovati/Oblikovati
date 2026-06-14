// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/health"
	"oblikovati.org/solve"
)

// publicHealth maps the host's health.Status onto the public types.HealthStatus the
// contract and wire surfaces report. The two vocabularies are kept value-aligned so the
// mapping is total.
func publicHealth(s health.Status) types.HealthStatus {
	switch s {
	case health.Warning:
		return types.HealthWarning
	case health.Sick:
		return types.HealthSick
	case health.Suppressed:
		return types.HealthSuppressed
	default:
		return types.HealthOK
	}
}

// healthFromStatus maps a solve status onto an assembly's overall health: redundant or
// conflicting constraints are a warning (still solvable), everything else is healthy.
// Non-convergence is surfaced separately on the solve result.
func healthFromStatus(s solve.Status) types.HealthStatus {
	if s == solve.OverConstrained {
		return types.HealthWarning
	}
	return types.HealthOK
}
