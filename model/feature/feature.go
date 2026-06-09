// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"sync/atomic"

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
	Keys       *identity.KeyManager
	SourceTool func(id ID) (tool *topo.Body, op ops.PartFeatureOperation, ok bool)
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

// Output is what a feature produces: the new running body state.
type Output struct {
	Bodies []*topo.Body
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
