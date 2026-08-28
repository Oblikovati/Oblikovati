// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
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
	items        []*PartFeature
	byID         map[ID]*PartFeature
	params       *param.Parameters
	eop          int
	result       []*topo.Body
	resources    ResourceStore
	fonts        text.FontResolver
	workingScale func() float64              // ADR-0042 Phase 2: live working scale (cm per working unit) for re-import
	relief       func() ReliefSpec           // the sheet-metal style's bend relief, read live (#2072)
	corner       func() CornerReliefSpec     // and its corner relief
	transition   func() types.BendTransition // and its bend transition (#1959)
	miterGap     func() float64              // and the gap it mitres corners with (#1961)
}

// ResourceStore reads embedded imported-file bytes by their document resource UUID
// (ADR-0031). The owning content (model/compdef) implements it and wires it into the engine
// so an imported-body feature can re-derive its body from the document instead of from disk.
type ResourceStore interface {
	ResourceBytes(id string) ([]byte, bool)
}

// SetReliefSpec wires the sheet-metal style's bend relief into the engine, read live so a style
// edit repropagates to every relieved bend (#2072). The owning content sets it when the part
// enters the sheet-metal environment; a part that never does simply has none.
func (fs *PartFeatures) SetReliefSpec(f func() ReliefSpec) { fs.relief = f }

// SetCornerReliefSpec wires the style's CORNER relief in, read live like the bend relief (#2072).
func (fs *PartFeatures) SetCornerReliefSpec(f func() CornerReliefSpec) { fs.corner = f }

// SetBendTransition wires the style's bend transition in, read live (#1959).
func (fs *PartFeatures) SetBendTransition(f func() types.BendTransition) { fs.transition = f }

// bendTransition reads the current transition; with no style there is nothing to shape.
func (fs *PartFeatures) bendTransition() types.BendTransition {
	if fs.transition == nil {
		return types.NoBendTransition
	}
	return fs.transition()
}

// SetMiterGap wires the style's miter gap in, read live (#1961).
func (fs *PartFeatures) SetMiterGap(f func() float64) { fs.miterGap = f }

// miterGapOf reads the current miter gap; with no style there is none.
func (fs *PartFeatures) miterGapOf() float64 {
	if fs.miterGap == nil {
		return 0
	}
	return fs.miterGap()
}

// cornerReliefSpec reads the current corner relief. With no style there is nothing to cut, which
// the tear shape says exactly.
func (fs *PartFeatures) cornerReliefSpec() CornerReliefSpec {
	if fs.corner == nil {
		return CornerReliefSpec{Shape: types.CornerTear}
	}
	return fs.corner()
}

// bendsBefore collects the bends every feature ahead of pf already placed (#2072). A wall only
// meets another wall at a CORNER, and a corner needs both bends — so the feature that builds the
// second one is the first point at which the junction exists to be relieved.
func (fs *PartFeatures) bendsBefore(pf *PartFeature) []BendPlacement {
	var out []BendPlacement
	for _, item := range fs.items {
		if item == pf {
			return out
		}
		if item.suppress || !item.health.OK() {
			continue
		}
		if placed, ok := item.Definition().(PlacedBend); ok {
			if p, ok := placed.Placement(); ok {
				out = append(out, p)
			}
		}
	}
	return out
}

// reliefSpec reads the current relief, or the zero spec (which cuts nothing) for a part with no
// sheet-metal style.
func (fs *PartFeatures) reliefSpec() ReliefSpec {
	if fs.relief == nil {
		return ReliefSpec{}
	}
	return fs.relief()
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
