// SPDX-License-Identifier: GPL-2.0-only

// Package depend holds the document-free primitives of incremental recompute: the
// opaque identity of anything a recompute can read (a Key) and the set-membership used to
// decide which consumers a change dirties. It is the foundation seam described in ADR-0044
// — the engine that today attributes parameter edits to features, and that a future
// cross-part adaptive reference plugs into by producing a new Kind of Key, without
// reshaping the attribution logic.
//
// It imports nothing from the rest of the model on purpose: param, sketch, feature, and
// compdef all depend on it, never the reverse, so the dependency points toward this stable
// core (the dependency rule, ADR-0044).
package depend

// Key is the opaque identity of one thing a recompute reads. The producer assigns ID
// within its Kind; an ID is only ever compared for equality against another Key of the
// SAME Kind, never interpreted, so two kinds may freely reuse the same numeric ID space.
// Key is a comparable value type so it serves directly as a map key and in set membership.
type Key struct {
	Kind KeyKind
	ID   uint64
}

// KeyKind is the category of thing a Key identifies. The set is closed and small by
// design: a new kind is added only when a genuinely new SOURCE of change appears (today
// parameters; tomorrow external geometry), and adding one must not require touching the
// attribution logic — that is the property ADR-0044 protects.
type KeyKind uint8

const (
	// ParameterKey identifies a model parameter (param.ID widened to uint64). It is the
	// only kind produced today: every footprint and change-set the engine builds is
	// parameter-keyed.
	ParameterKey KeyKind = iota

	// ExternalGeometryKey identifies a geometric entity (a face/edge) in ANOTHER document,
	// resolved by the orchestrator (app.Session) against the workspace reference graph and
	// given a stable identity by the external geometric reference of ADR-0040. It is
	// reserved for cross-part adaptive references and has no producer yet (ADR-0044): the
	// attribution machinery is already kind-agnostic, so adaptivity supplies the producer
	// without changing the engine.
	ExternalGeometryKey
)
