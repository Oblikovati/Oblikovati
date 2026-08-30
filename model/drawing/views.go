// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/collview"
)

// Drawing view generation — the COLLECTION and SHARED PROJECTION / HLR invocation (M48 #2227 split of
// views.go). This file holds the DrawingViews collection, the recompute (associativity) pass, the
// per-view projection-frame dispatch (basisFor), and the label/style edits. The per-kind adders and
// previews live in views_base.go, views_section.go, views_detail.go and views_derived.go.

// bodyLookup resolves the drawing's referenced model to its B-rep body for projection. It is
// the seam between this package (which knows nothing of the workspace) and the host, which
// finds the referenced model document and returns its body. A nil hook (no resolver wired) or
// an unresolved reference yields (nil, false), so view creation reports a clear error.
type bodyLookup func() (*topo.Body, bool)

// DrawingViews is a sheet's ordered, named view collection. It holds the body-resolution hook
// so it can project the referenced model when a view is added or recomputed.
type DrawingViews struct {
	items []*DrawingView
	body  bodyLookup
}

func newDrawingViews(body bodyLookup) *DrawingViews { return &DrawingViews{body: body} }

// parentBaseFor resolves a derived view's parent (which must be a base view) and the referenced
// model body — the common precondition for auxiliary/section/detail views.
func (vs *DrawingViews) parentBaseFor(parentView string) (*DrawingView, *topo.Body, error) {
	parent, ok := vs.ByName(parentView)
	if !ok {
		return nil, nil, fmt.Errorf("drawing: no parent view %q to derive from", parentView)
	}
	if parent.viewType != types.DrawingViewBase {
		return nil, nil, fmt.Errorf("drawing: %q is not a base view; derive from a base view", parentView)
	}
	body, ok := vs.resolveBody()
	if !ok {
		return nil, nil, fmt.Errorf("drawing: no referenced model to project")
	}
	return parent, body, nil
}

// Recompute re-projects every view against the current referenced model — the associativity
// path after a model edit. Draft views (no model) refresh their frame regardless; the
// model-backed views are left untouched when no model resolves.
func (vs *DrawingViews) Recompute() {
	vs.resolveEffectiveStyles()
	vs.applyAlignments() // pull locked views onto their anchors before projecting (#1988)
	body, ok := vs.resolveBody()
	for _, v := range vs.items {
		if v.viewType == types.DrawingViewDraft {
			v.recomputeDraftFrame()
			continue
		}
		if !ok {
			continue
		}
		if basis, ok := vs.basisFor(v, bodyCenter(body)); ok {
			v.recompute(body, basis)
		}
	}
}

// ViewLabelStyle carries the optional view-label overrides to apply (#1983); a nil field is left
// unchanged. XMM and YMM are applied together (both must be set to reposition the caption).
type ViewLabelStyle struct {
	Text      *string
	ShowLabel *bool
	ShowName  *bool
	ShowScale *bool
	XMM       *float64
	YMM       *float64
}

// SetLabel applies the given label overrides to the named view (#1983), erroring when no view
// carries that name.
func (vs *DrawingViews) SetLabel(name string, style ViewLabelStyle) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no view named %q", name)
	}
	if style.Text != nil {
		v.SetLabelText(*style.Text)
	}
	if style.ShowLabel != nil {
		v.SetShowLabel(*style.ShowLabel)
	}
	if style.ShowName != nil {
		v.SetShowName(*style.ShowName)
	}
	if style.ShowScale != nil {
		v.SetShowScale(*style.ShowScale)
	}
	if style.XMM != nil && style.YMM != nil {
		v.SetLabelPositionMM(*style.XMM, *style.YMM)
	}
	return nil
}

// SetDisplayTangentEdges shows/hides the named view's smooth tangent edges and re-projects it so the
// change takes effect immediately (#1984).
func (vs *DrawingViews) SetDisplayTangentEdges(name string, show bool) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no view named %q", name)
	}
	v.SetDisplayTangentEdges(show)
	vs.Recompute()
	return nil
}

// resolveEffectiveStyles resolves each view's FromBase style to its base view's style before the
// projection pass, so a derived view renders with its parent's style associatively (#1985).
func (vs *DrawingViews) resolveEffectiveStyles() {
	lookup := func(name string) (types.DrawingViewStyle, bool) {
		if base, ok := vs.ByName(name); ok {
			return base.style, true
		}
		return 0, false
	}
	for _, v := range vs.items {
		v.resolveEffectiveStyle(lookup)
	}
}

// basisFor returns the projection frame a view uses, dispatching on its type. A base view
// projects its standard orientation; a derived view derives from its parent's frame (and yields
// ok=false if that parent is missing, so it is left as-is rather than mis-projected).
func (vs *DrawingViews) basisFor(v *DrawingView, origin math.Point3) (hlr.View, bool) {
	if v.viewType == types.DrawingViewBase {
		return baseBasis(v.orientation, origin), true
	}
	base, ok := vs.ByName(v.baseView)
	if !ok {
		return hlr.View{}, false
	}
	parent := baseBasis(base.orientation, origin)
	switch v.viewType {
	case types.DrawingViewProjected:
		return projectedBasis(parent, v.direction, origin), true
	case types.DrawingViewAuxiliary:
		return auxiliaryBasis(parent, v.foldAngle, origin), true
	case types.DrawingViewSection, types.DrawingViewSlice:
		return sectionBasis(parent, v.section, base.scale, base.centerX, base.centerY, origin), true
	default: // detail/breakout/break reuse the parent projection (recompute clips/breaks); re-map
		// their sheet-mm region against the parent's current placement so they track a rescale/move.
		v.refreshParentRegion(base)
		return parent, true
	}
}

// refreshParentRegion re-derives a detail/break view's region (clip circle or break band) from
// its persisted sheet-mm definition against the parent's current placement — the associativity
// path when the parent is rescaled or moved.
func (v *DrawingView) refreshParentRegion(parent *DrawingView) {
	switch v.viewType {
	case types.DrawingViewDetail, types.DrawingViewBreakout:
		v.detail = detailBoundaryOf(parent, v.detail.sheetCX, v.detail.sheetCY, v.detail.sheetR)
	case types.DrawingViewBreak:
		v.brk = breakBandOf(parent, v.brk.orientation, v.brk.sheetG0, v.brk.sheetG1)
	}
}

// Count, Item and ByName read the collection.
func (vs *DrawingViews) Count() int { return len(vs.items) }

func (vs *DrawingViews) Item(i int) *DrawingView { return collview.At(vs.items, i) }

func (vs *DrawingViews) ByName(name string) (*DrawingView, bool) {
	for _, v := range vs.items {
		if v.name == name {
			return v, true
		}
	}
	return nil, false
}

// Remove deletes the named view; removing a base view also removes the views derived from it
// (projected/auxiliary/…), which have no parent left to derive from.
func (vs *DrawingViews) Remove(name string) error {
	if _, ok := vs.ByName(name); !ok {
		return fmt.Errorf(errNoViewNamed, name)
	}
	kept := vs.items[:0]
	for _, v := range vs.items {
		if v.name == name || v.baseView == name {
			continue
		}
		kept = append(kept, v)
	}
	vs.items = kept
	return nil
}

func (vs *DrawingViews) resolveBody() (*topo.Body, bool) {
	if vs.body == nil {
		return nil, false
	}
	return vs.body()
}

// uniqueName returns the requested name (erroring if taken) or the next auto "VIEW:N".
func (vs *DrawingViews) uniqueName(requested string) (string, error) {
	if requested == "" {
		return vs.nextName(), nil
	}
	if _, exists := vs.ByName(requested); exists {
		return "", fmt.Errorf("drawing: view %q already exists", requested)
	}
	return requested, nil
}

func (vs *DrawingViews) nextName() string {
	for n := len(vs.items) + 1; ; n++ {
		name := fmt.Sprintf("VIEW:%d", n)
		if _, exists := vs.ByName(name); !exists {
			return name
		}
	}
}

func positiveScale(s float64) float64 {
	if s <= 0 {
		return 1
	}
	return s
}
