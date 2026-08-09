// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	"sync/atomic"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/param"
)

// ErrDeferred is returned by a feature whose inputs resolved successfully but whose
// B-rep generation is not yet implemented (a kernel phase not yet reached). The
// engine maps it to health.Warning with the body state passed through unchanged —
// so the feature's inputs are validated and tracked while honestly signaling that
// its geometry is pending, distinct from a Sick lost-reference failure.
var ErrDeferred = errors.New("feature generation deferred to a later kernel phase")

// ID is a feature's session-stable handle. Renaming a feature does not change it,
// so reorder and dependency edges remain valid across renames.
type ID uint64

var idSeq atomic.Uint64

func nextID() ID { return ID(idSeq.Add(1)) }

// Ref is a feature input: a reference key into topology, a sketch, or a work
// feature, resolved lazily at the top of recompute against the current model
// (ADR-0010). It is a value, not a pointer, so it survives the rebuild.
type Ref struct {
	Key identity.RefKey
}

// Input is what the engine hands a feature on recompute: the running body state
// (the result of the clean prefix), the parameter set, and the key manager for
// resolving [Ref] inputs.
//
// SourceTool lets a feature that replicates another (a pattern/mirror) recover the
// geometric contribution of a source feature: the tool body it added or removed and the
// boolean operation it used, computed from the source's cached before/after body state. A
// pattern uses it to re-apply a cut/join at each occurrence (so patterning a hole cuts N
// holes in one body) instead of duplicating the whole body. It is nil for features the
// engine does not resolve sources for; ok is false when the id is unknown or has no delta.
type Input struct {
	Bodies     []*topo.Body
	Params     *param.Parameters
	SourceTool func(id ID) (tool *topo.Body, op ops.PartFeatureOperation, ok bool)
	// Diag collects the kernel diagnostics raised while the feature rebuilds — the facet/CSG
	// fallback defects a boolean records when it abandons the analytic path (#1601). The engine
	// installs a fresh recorder per evaluation and stores what it collected on the PartFeature;
	// a nil recorder is a valid sink (diag.Recorder is nil-safe), so preview paths that do not
	// consume diagnostics pass nothing.
	Diag *diag.Recorder
	// Relief is the sheet-metal style's bend relief, resolved for this recompute (#2072). The
	// sizes could ride on parameters like the thickness does, but the SHAPE is not a number, so
	// the whole spec arrives together rather than half here and half through Params.
	Relief ReliefSpec
	// PriorBends are the bends placed by the features ahead of this one (#2072). A wall meets
	// another wall only at a corner, so relieving that corner needs both bends — and the feature
	// building the second one is where they first both exist.
	PriorBends []BendPlacement
	// Corner is the sheet-metal style's CORNER relief, resolved for this recompute (#2072).
	Corner CornerReliefSpec
	// Transition is the style's bend transition (#1959), which a feature's own bend options may
	// override.
	Transition types.BendTransition
}

// OperationalFeature is a feature that applies a boolean operation against the running
// bodies (extrude, the modify booleans, …). A pattern reads the source's operation through
// it to decide how to replicate the source (cut/join/intersect vs a new-body copy).
type OperationalFeature interface {
	Operation() ops.PartFeatureOperation
}

// ToolFeature is an [OperationalFeature] that can hand a pattern the exact tool body it
// combined (its prism/sweep solid). Replicating that clean tool is more robust than diffing
// the running bodies — the difference can degenerate on curved geometry.
type ToolFeature interface {
	OperationalFeature
	ToolBody() *topo.Body
}

// Output is what a feature produces: the new running body state, plus any reference
// heals that occurred while resolving its inputs (ADR-0043 P6). Heals ride with the
// Output — they are part of the recompute result, not an error — and the engine
// turns a non-empty Heals into health.Warning while keeping the rebuilt body.
type Output struct {
	Bodies []*topo.Body
	Heals  []ReferenceHeal
}

// ReferenceHeal records that a stored reference key did not match any entity exactly
// but was recovered by a degraded tier of the topological binder (ADR-0043 P6): the
// feature still rebuilds on the recovered entity, but the drift is surfaced as a
// Warning so the user can re-pick if the recovery was not what they meant.
type ReferenceHeal struct {
	Key   []byte             // the stored reference key whose exact entity was gone
	Match identity.MatchType // the tier that recovered it (ancestral or geometric)
}

// healReason renders one or more reference heals as a single Warning message.
func healReason(heals []ReferenceHeal) string {
	if len(heals) == 1 {
		return fmt.Sprintf("reference %q healed (%s match) — re-pick to confirm", keyText(heals[0].Key), heals[0].Match)
	}
	return fmt.Sprintf("%d references healed by degraded binding — re-pick to confirm", len(heals))
}

// Feature is one recipe in the history. Recompute is pure: it reads the input and
// returns the new body state, or an error if it cannot evaluate (a lost reference,
// a failed modeling op) — the engine turns that error into [health.Sick] without
// aborting the rebuild.
type Feature interface {
	// Kind names the feature type (e.g. "extrude") for diagnostics and UI.
	Kind() string
	// Recompute evaluates the feature against the running state.
	Recompute(in Input) (Output, error)
}
