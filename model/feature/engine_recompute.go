// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/health"
)

// Feature recompute engine — the RECOMPUTE LOOP and HEALTH/QUARANTINE (M48 #2234 split of engine.go).
// Walks the feature list from the earliest dirty feature, evaluates each against the running bodies,
// catches a panicking or erroring feature and marks it Sick (quarantining its dependents), and derives
// each feature's source tool/delta. The feature graph/ordering lives in engine_graph.go; the collection
// and configuration in engine.go.

// Recompute replays the dirty tail: it finds the earliest dirty feature, reuses the
// cached body state before it, and evaluates forward to the end-of-part marker.
// Failures become feature health (sick) and poison dependents; the rebuild never
// aborts.
func (fs *PartFeatures) Recompute() {
	end := fs.effectiveEnd()
	if start := fs.earliestDirty(end); start < 0 {
		// Nothing dirty: the result is the cached body state at the cutoff. Re-deriving
		// it (rather than leaving fs.result untouched) keeps the result correct after a
		// Remove that shortened the program — the deleted tail no longer contributes.
		fs.result = fs.prefixBodies(end)
	} else {
		fs.result = fs.evaluateFrom(start, end)
	}
	fs.fileResultBodies(end) // whose body is whose, so Diagnostics() can report what it CARRIES (#2058)
}

// evaluateFrom replays the program from the first dirty feature to the cutoff, threading the running
// body state through each and poisoning the dependents of any that sickens.
func (fs *PartFeatures) evaluateFrom(start, end int) []*topo.Body {
	bodies := fs.prefixBodies(start)
	sick := fs.sickBefore(start)
	for i := start; i < end; i++ {
		bodies = fs.evaluate(fs.items[i], bodies, sick)
	}
	return bodies
}

// PreviewResult evaluates a candidate feature as if it were appended at the end-of-part
// marker, against the cached prefix bodies, and returns the resulting body state — WITHOUT
// mutating the program (no Add, no dirty flags, no fs.result change, no events). It is the
// non-destructive seam behind the live in-canvas feature preview: a tool builds its draft
// feature and asks "what would the model look like?" once per change, reusing the clean
// prefix the engine already cached, so a preview costs ~one feature recompute.
//
// Example: ext := buildExtrudeFeature(def); bodies, err := part.Features().PreviewResult(ext).
func (fs *PartFeatures) PreviewResult(candidate Feature) ([]*topo.Body, error) {
	if candidate == nil {
		return nil, errors.New("feature: PreviewResult got a nil candidate")
	}
	bodies := fs.prefixBodies(fs.effectiveEnd())
	out, err := candidate.Recompute(Input{Bodies: bodies, Params: fs.params, SourceTool: fs.sourceTool, SourceTools: fs.sourceTools, Relief: fs.reliefSpec()})
	if err != nil {
		return nil, err
	}
	return out.Bodies, nil
}

// evaluate runs one feature, updating its health/cache and returning the running
// body state after it. It captures the model parameters the feature read directly
// (its suppression condition and its own recompute, e.g. a sheet-metal thickness)
// into pf.paramReads, so a later parameter edit can skip the feature when it touches
// none of them (Oblikovati#1414).
func (fs *PartFeatures) evaluate(pf *PartFeature, bodies []*topo.Body, sick map[ID]bool) []*topo.Body {
	if fs.params == nil {
		return fs.evaluateBody(pf, bodies, sick)
	}
	var out []*topo.Body
	pf.paramReads = fs.params.TrackKeys(func() { out = fs.evaluateBody(pf, bodies, sick) })
	return out
}

// evaluateBody is evaluate without the parameter-read capture (see evaluate).
func (fs *PartFeatures) evaluateBody(pf *PartFeature, bodies []*topo.Body, sick map[ID]bool) []*topo.Body {
	pf.dirty = false
	if pf.suppress || (pf.condition != nil && pf.condition.holds(fs.params)) {
		pf.health = health.Health{Status: health.Suppressed}
		pf.cached = bodies
		return bodies // suppressed features pass the body state through unchanged
	}
	if fs.dependsOnSick(pf, sick) {
		pf.health = health.Sicken("upstream feature is sick")
		sick[pf.id] = true
		pf.cached = bodies
		return bodies
	}
	pf.recomputes++
	rec := &diag.Recorder{}
	out, err := safeRecompute(pf, Input{Bodies: bodies, Params: fs.params, SourceTool: fs.sourceTool, SourceTools: fs.sourceTools,
		Diag: rec, Relief: fs.reliefSpec(), Corner: fs.cornerReliefSpec(), Transition: fs.bendTransition(), MiterGap: fs.miterGapOf(), PriorBends: fs.bendsBefore(pf)})
	pf.diags = rec.Records()
	return fs.classify(pf, bodies, out, err, sick)
}

// safeRecompute runs one feature's Recompute, converting a PANIC (a nil-deref or
// index-out-of-range the alpha kernel can hit on an unhandled boolean/tessellation case) into
// an ordinary error. That error flows through classify like any other failure, so the feature
// goes Sick and the rebuild continues its tail instead of the panic unwinding the whole
// Recompute and crashing the app — the canonical "a sick feature never aborts the rebuild"
// rule (Oblikovati#1415). The recover is scoped to this single call so a programmer error
// anywhere else still surfaces. The error names the offending feature (via classify's Kind
// prefix) and the recovered panic value.
func safeRecompute(pf *PartFeature, in Input) (out Output, err error) {
	defer func() {
		if r := recover(); r != nil {
			out, err = Output{}, fmt.Errorf("panicked during recompute: %v", r)
		}
	}()
	return pf.feature.Recompute(in)
}

// sourceTool resolves a source feature's geometric contribution for a pattern/mirror: the
// tool body it added or removed (the before/after delta) and the operation it applied. A
// new-body or undeterminable source returns ok=false, so the replicator copies the whole
// body instead (the right behavior for placing independent solids).
func (fs *PartFeatures) sourceTool(id ID) (*topo.Body, ops.PartFeatureOperation, bool) {
	idx := -1
	for i, it := range fs.items {
		if it.id == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, ops.NewBody, false
	}
	f := fs.items[idx].feature
	op := operationOf(f)
	// Prefer the source feature's own tool (a clean prism) over a before/after difference,
	// which can degenerate on curved geometry.
	if tf, ok := f.(ToolFeature); ok {
		if tool := tf.ToolBody(); tool != nil {
			return tool, op, true
		}
	}
	tool, derr := sourceDelta(lastSolid(fs.prefixBodies(idx)), lastSolid(fs.items[idx].cached), op)
	if derr != nil || tool == nil {
		return nil, op, false
	}
	return tool, op, true
}

// sourceTools is sourceTool's multi-tool sibling (#2066): a source feature that applies several
// booleans (a from-plane emboss raises and cuts about its plane) reports each here so a pattern
// replicates all of them. ok is false — and the caller falls back to the single sourceTool — for a
// feature that applies one tool or whose multi-tool set is empty this recompute.
func (fs *PartFeatures) sourceTools(id ID) ([]ToolApplication, bool) {
	for _, it := range fs.items {
		if it.id != id {
			continue
		}
		if mtf, ok := it.feature.(MultiToolFeature); ok {
			if apps := mtf.ToolApplications(); len(apps) > 0 {
				return apps, true
			}
		}
		return nil, false
	}
	return nil, false
}

// operationOf returns a feature's boolean operation, or NewBody when it does not apply one
// (so its replication falls back to copying whole bodies).
func operationOf(f Feature) ops.PartFeatureOperation {
	if of, ok := f.(OperationalFeature); ok {
		return of.Operation()
	}
	return ops.NewBody
}

// sourceDelta returns the material a feature contributed: for a cut, before−after (the
// removed chunk, the tool clipped to the body); for join/intersect, after−before (the added
// chunk). Its booleans are internal DIFFING of already-built state, not a user operation on
// the model, so they deliberately run without a diagnostics recorder (#1601): a facet
// fallback here degrades only the replicated tool a pattern derives, and the pattern's own
// boolean records against the pattern feature. NewBody, an absent state, or a feature that contributed nothing (a deferred feature,
// before == after → an empty difference) yields no delta, so the caller adds nothing for that
// source rather than copying the whole body.
func sourceDelta(before, after *topo.Body, op ops.PartFeatureOperation) (*topo.Body, error) {
	switch op {
	case ops.Cut:
		if before == nil || after == nil {
			return nil, nil
		}
		d, err := ops.Boolean(ops.Cut, before, after)
		return nonEmpty(d), err
	case ops.Join, ops.Intersect:
		if after == nil {
			return nil, nil
		}
		if before == nil {
			return after, nil
		}
		d, err := ops.Boolean(ops.Cut, after, before)
		return nonEmpty(d), err
	default:
		return nil, nil
	}
}

// nonEmpty returns b unless it is an empty body (no faces) — an empty difference means the
// source contributed nothing to replicate, so the pattern adds nothing rather than copying.
func nonEmpty(b *topo.Body) *topo.Body {
	if b == nil || len(b.Faces()) == 0 {
		return nil
	}
	return b
}

// lastBody returns the last body of a running state (the one a feature's boolean acts on),
// or nil when empty.
func lastSolid(bodies []*topo.Body) *topo.Body {
	if len(bodies) == 0 {
		return nil
	}
	return bodies[len(bodies)-1]
}

// sickReason prefixes a feature's failure with its kind for the browser, unless the error already
// leads with that kind — many kernel ops prefix their own errors with the same word (e.g. a fillet
// op returns "fillet: …"), which otherwise produced a "fillet: fillet: …" double prefix.
func sickReason(kind string, err error) string {
	msg := err.Error()
	if strings.HasPrefix(msg, kind+":") {
		return msg
	}
	return fmt.Sprintf("%s: %s", kind, msg)
}

// classify turns a feature's recompute result into health + the running body state:
// ErrDeferred → warning (passthrough); other error → sick (poison); a healed
// reference (ADR-0043 P6) → warning with the rebuilt body kept; nil → healthy.
func (fs *PartFeatures) classify(pf *PartFeature, bodies []*topo.Body, out Output, err error, sick map[ID]bool) []*topo.Body {
	switch {
	case errors.Is(err, ErrDeferred):
		pf.health = health.Health{Status: health.Warning, Reason: err.Error()}
		pf.cached = out.Bodies
	case err != nil:
		pf.health = health.Sicken(sickReason(pf.Kind(), err))
		sick[pf.id] = true
		pf.cached = bodies
	case len(out.Heals) > 0:
		// The feature rebuilt successfully, but one or more references bound through a
		// degraded tier instead of an exact match — keep the rebuilt body and flag the
		// drift so the user can re-pick, rather than reporting a clean recompute.
		pf.health = health.Health{Status: health.Warning, Reason: healReason(out.Heals)}
		pf.cached = out.Bodies
	default:
		pf.health = health.Healthy
		pf.cached = out.Bodies
	}
	return pf.cached
}
