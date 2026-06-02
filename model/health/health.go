// SPDX-License-Identifier: GPL-2.0-only

// Package health is the shared health vocabulary for modeling entities — the
// status a feature, dimension, mate or reference reports when it cannot be fully
// evaluated. It is the modernized HealthStatusEnum (architecture core/02): a
// recompute or bind failure becomes health state on the entity, never a panic
// (parametric-cad §2).
//
// It is deliberately tiny and dependency-free so every modeling package can adopt
// the same vocabulary without coupling. (The param package predates this and keeps
// its own evaluation-specific status; the two converge as features land.)
package health

// Status is the condition of a modeling entity.
type Status uint8

const (
	// OK: the entity is fully evaluated and valid.
	OK Status = iota
	// Warning: usable but flagged (e.g. an over-constrained sketch that still solved).
	Warning
	// Sick: cannot be evaluated and needs user attention (e.g. a lost reference).
	Sick
	// Suppressed: intentionally excluded from evaluation by the user.
	Suppressed
)

// String returns a stable lowercase name for diagnostics.
func (s Status) String() string {
	switch s {
	case OK:
		return "ok"
	case Warning:
		return "warning"
	case Sick:
		return "sick"
	case Suppressed:
		return "suppressed"
	default:
		return "unknown"
	}
}

// Health is a status plus a human-readable reason when the status is not OK. The
// reason names the offending cause so the UI can surface it for repair.
type Health struct {
	Status Status
	Reason string
}

// Healthy is the zero-value healthy state.
var Healthy = Health{Status: OK}

// OK reports whether the entity is fully healthy.
func (h Health) OK() bool { return h.Status == OK }

// Sicken returns a Sick health carrying reason.
func Sicken(reason string) Health {
	return Health{Status: Sick, Reason: reason}
}
