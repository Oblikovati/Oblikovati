// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/identity"
	"github.com/Oblikovati/oblikovati/model/param"
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
	keys   *identity.KeyManager
	eop    int
	result []*topo.Body
}

// NewPartFeatures creates an empty feature program. params drives expressions and
// conditional suppression; keys resolves topology input refs (either may be nil
// for simple cases).
func NewPartFeatures(params *param.Parameters, keys *identity.KeyManager) *PartFeatures {
	return &PartFeatures{byID: map[ID]*PartFeature{}, params: params, keys: keys, eop: eopAll}
}

// Add appends a feature (initially dirty) with its dependency feature ids.
func (fs *PartFeatures) Add(f Feature, deps ...ID) *PartFeature {
	pf := &PartFeature{id: nextID(), name: f.Kind(), feature: f, deps: deps, dirty: true, health: health.Healthy}
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

// Recompute replays the dirty tail: it finds the earliest dirty feature, reuses the
// cached body state before it, and evaluates forward to the end-of-part marker.
// Failures become feature health (sick) and poison dependents; the rebuild never
// aborts.
func (fs *PartFeatures) Recompute() {
	end := fs.effectiveEnd()
	start := fs.earliestDirty(end)
	if start < 0 {
		// Nothing dirty: the result is the cached body state at the cutoff. Re-deriving
		// it (rather than leaving fs.result untouched) keeps the result correct after a
		// Remove that shortened the program — the deleted tail no longer contributes.
		fs.result = fs.prefixBodies(end)
		return
	}
	bodies := fs.prefixBodies(start)
	sick := fs.sickBefore(start)
	for i := start; i < end; i++ {
		bodies = fs.evaluate(fs.items[i], bodies, sick)
	}
	fs.result = bodies
}

// evaluate runs one feature, updating its health/cache and returning the running
// body state after it.
func (fs *PartFeatures) evaluate(pf *PartFeature, bodies []*topo.Body, sick map[ID]bool) []*topo.Body {
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
	out, err := pf.feature.Recompute(Input{Bodies: bodies, Params: fs.params, Keys: fs.keys})
	return fs.classify(pf, bodies, out, err, sick)
}

// classify turns a feature's recompute result into health + the running body state:
// ErrDeferred → warning (passthrough), other error → sick (poison), nil → healthy.
func (fs *PartFeatures) classify(pf *PartFeature, bodies []*topo.Body, out Output, err error, sick map[ID]bool) []*topo.Body {
	switch {
	case errors.Is(err, ErrDeferred):
		pf.health = health.Health{Status: health.Warning, Reason: err.Error()}
		pf.cached = out.Bodies
	case err != nil:
		pf.health = health.Sicken(fmt.Sprintf("%s: %v", pf.Kind(), err))
		sick[pf.id] = true
		pf.cached = bodies
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
