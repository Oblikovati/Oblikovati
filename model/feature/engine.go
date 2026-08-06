// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/health"
	"oblikovati.org/model/param"
	"oblikovati.org/model/seq"
	"oblikovati.org/model/text"
)

// eopAll means the end-of-part marker is at the end (evaluate the whole program).
const eopAll = -1

// PartFeatures is the ordered feature program and its recompute engine — the
// PartFeatures collection. It evaluates the dirty tail in history order, reusing
// the clean prefix, isolates failures as feature health, and supports
// reorder/rename/suppression and the end-of-part marker (ADR-0010).
type PartFeatures struct {
	items  []*PartFeature
	byID   map[ID]*PartFeature
	params *param.Parameters
	eop    int
	result []*topo.Body
	// resultDiags is what each body in result CARRIES (its build report, its mesh report), kept
	// keyed on body identity so an unchanged body is judged once and not re-meshed (#2058).
	resultDiags  map[*topo.Body][]diag.Diagnostic
	resources    ResourceStore
	fonts        text.FontResolver
	workingScale func() float64 // ADR-0042 Phase 2: live working scale (cm per working unit) for re-import
}

// ResourceStore reads embedded imported-file bytes by their document resource UUID
// (ADR-0031). The owning content (model/compdef) implements it and wires it into the engine
// so an imported-body feature can re-derive its body from the document instead of from disk.
type ResourceStore interface {
	ResourceBytes(id string) ([]byte, bool)
}

// SetResourceStore wires the document's resource table into the engine so feature restore can
// read imported files by UUID. Set by the owning content after construction (the engine is
// recreated on a recipe reset, so this must be re-wired each time).
func (fs *PartFeatures) SetResourceStore(rs ResourceStore) { fs.resources = rs }

// SetWorkingScaleResolver wires a getter for the live working scale (centimetres per working
// length unit), so re-importing an embedded foreign file on reopen scales it into the
// document's working coordinates (ADR-0042 Phase 2). Like the resource store it is re-wired
// after a recipe reset. When unset, re-import falls back to the centimetre default.
func (fs *PartFeatures) SetWorkingScaleResolver(fn func() float64) { fs.workingScale = fn }

// workingTargetMM resolves the working-unit millimetre size for a re-import (the working scale
// in centimetres × the centimetre's millimetre size), or 0 when no resolver is wired
// (ImportBodiesFromData then applies the centimetre default).
func (fs *PartFeatures) workingTargetMM() float64 {
	if fs.workingScale == nil {
		return 0
	}
	return fs.workingScale() * exchange.DBUnitMM
}

// SetFontResolver wires the document's font resolver (resource-aware) into the engine so a
// text/emboss feature resolves its font from the document's embedded/app-provided faces. Like
// the resource store, it is re-wired after a recipe reset (the engine is recreated).
func (fs *PartFeatures) SetFontResolver(r text.FontResolver) { fs.fonts = r }

// FontResolver returns the engine's document font resolver, or the embedded-faces default when
// none was wired (a bare engine still resolves plain family-named text).
func (fs *PartFeatures) FontResolver() text.FontResolver {
	if fs.fonts != nil {
		return fs.fonts
	}
	return text.DefaultResolver()
}

// NewPartFeatures creates an empty feature program. params drives expressions and
// conditional suppression; keys resolves topology input refs (either may be nil
// for simple cases).
func NewPartFeatures(params *param.Parameters) *PartFeatures {
	return &PartFeatures{byID: map[ID]*PartFeature{}, params: params, eop: eopAll}
}

// Add appends a feature (initially dirty) with its dependency feature ids.
func (fs *PartFeatures) Add(f Feature, deps ...ID) *PartFeature {
	pf := &PartFeature{id: nextID(), name: f.Kind(), feature: f, deps: deps, dirty: true, health: health.Healthy, seq: seq.Next()}
	fs.items = append(fs.items, pf)
	fs.byID[pf.id] = pf
	return pf
}

// Remove deletes the feature with the given id from the program, reporting whether it
// was found. The tail from the removed position is marked dirty so the next Recompute
// rebuilds the body state without it (the clean prefix before it is still reused).
func (fs *PartFeatures) Remove(id ID) bool {
	if _, ok := fs.byID[id]; !ok {
		return false
	}
	delete(fs.byID, id)
	for i, pf := range fs.items {
		if pf.id != id {
			continue
		}
		fs.items = append(fs.items[:i], fs.items[i+1:]...)
		for _, tail := range fs.items[i:] {
			tail.dirty = true
		}
		break
	}
	return true
}

// Count returns the number of features; Item returns the i-th in history order.
func (fs *PartFeatures) Count() int              { return len(fs.items) }
func (fs *PartFeatures) Item(i int) *PartFeature { return fs.items[i] }

// ByID returns the feature with the given id.
func (fs *PartFeatures) ByID(id ID) (*PartFeature, bool) { f, ok := fs.byID[id]; return f, ok }

// ByName returns the first feature with the given name.
func (fs *PartFeatures) ByName(name string) (*PartFeature, bool) {
	for _, f := range fs.items {
		if f.name == name {
			return f, true
		}
	}
	return nil, false
}

// UniqueName returns base suffixed with the smallest positive integer that is not
// already a feature name (Inventor numbers features Extrusion1, Extrusion2, …). A
// caller names a newly added feature with this so the browser shows distinct rows and
// Dear ImGui derives a distinct id per node — two nodes sharing a label trips ImGui's
// duplicate-id assertion on hover.
func (fs *PartFeatures) UniqueName(base string) string {
	for n := 1; ; n++ {
		name := base + strconv.Itoa(n)
		if _, taken := fs.ByName(name); !taken {
			return name
		}
	}
}

// Result returns the body state after evaluating up to the end-of-part marker.
func (fs *PartFeatures) Result() []*topo.Body { return fs.result }

// MarkDirty flags a feature for re-evaluation on the next recompute.
func (fs *PartFeatures) MarkDirty(f *PartFeature) { f.dirty = true }

// RequiresUpdate reports whether any feature up to the end-of-part marker is dirty — i.e. a
// recompute would change the result. It is the read-only "needs update" flag the API exposes
// (Inventor PartDocument.RequiresUpdate), the predicate behind documents.update (#139).
func (fs *PartFeatures) RequiresUpdate() bool { return fs.earliestDirty(fs.effectiveEnd()) >= 0 }

// MarkAllDirty flags every feature for re-evaluation. A parameter edit can change
// any feature's inputs — a dimension expression a sketch profile is built from, or
// a feature's own value closure (extrude distance, revolve angle) — but those
// inputs are read live and are not tracked as feature dependencies, so the engine
// would otherwise see nothing dirty and return the cached bodies. Invalidating the
// whole program on a parameter change keeps the model correct (a full parametric
// rebuild, as Inventor does).
func (fs *PartFeatures) MarkAllDirty() {
	for _, pf := range fs.items {
		pf.dirty = true
	}
}

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
	fs.diagnoseResultBodies(end) // what the surviving bodies CARRY, not just what the calls reported (#2058)
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
	out, err := candidate.Recompute(Input{Bodies: bodies, Params: fs.params, SourceTool: fs.sourceTool})
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
	out, err := safeRecompute(pf, Input{Bodies: bodies, Params: fs.params, SourceTool: fs.sourceTool, Diag: rec})
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

// effectiveEnd returns the evaluation cutoff (the EOP marker, or the full length).
func (fs *PartFeatures) effectiveEnd() int {
	if fs.eop == eopAll || fs.eop > len(fs.items) {
		return len(fs.items)
	}
	return fs.eop
}

// earliestDirty returns the index of the first dirty feature below end, or -1.
func (fs *PartFeatures) earliestDirty(end int) int {
	for i := 0; i < end; i++ {
		if fs.items[i].dirty {
			return i
		}
	}
	return -1
}

// prefixBodies returns the cached body state just before index start.
func (fs *PartFeatures) prefixBodies(start int) []*topo.Body {
	if start <= 0 {
		return nil
	}
	return fs.items[start-1].cached
}

// sickBefore collects the sick features in the reused clean prefix, so tail
// features depending on them are still poisoned.
func (fs *PartFeatures) sickBefore(start int) map[ID]bool {
	sick := map[ID]bool{}
	for i := 0; i < start; i++ {
		if fs.items[i].health.Status == health.Sick {
			sick[fs.items[i].id] = true
		}
	}
	return sick
}

// dependsOnSick reports whether any of pf's dependencies is currently sick.
func (fs *PartFeatures) dependsOnSick(pf *PartFeature, sick map[ID]bool) bool {
	for _, d := range pf.deps {
		if sick[d] {
			return true
		}
	}
	return false
}
