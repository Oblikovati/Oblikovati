// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
)

// View placement (#1988): a view can be rotated about its centre and locked to another view on a
// shared axis (so moving the anchor drags it) or freed. Rotation is applied in place(); the alignment
// lock is enforced at the start of each Recompute, before the views project.

// RotationDeg is the view's rotation about its centre, in degrees (CCW positive).
func (v *DrawingView) RotationDeg() float64 { return v.rotation * 180 / stdmath.Pi }

// SetRotationDeg sets the view's rotation about its centre (degrees, CCW positive).
func (v *DrawingView) SetRotationDeg(deg float64) { v.rotation = deg * stdmath.Pi / 180 }

// AlignedTo is the anchor view this view is locked to ("" when free).
func (v *DrawingView) AlignedTo() string { return v.alignedTo }

// Alignment is the shared-axis lock kind (horizontal/vertical/inPosition).
func (v *DrawingView) Alignment() types.DrawingViewAlignment { return v.alignment }

// IsAligned reports whether the view is locked to an anchor view.
func (v *DrawingView) IsAligned() bool {
	return v.alignedTo != "" && v.alignment != types.InPositionViewAlignment
}

// Justification is the view's centring mode (centered/fixed).
func (v *DrawingView) Justification() types.ViewJustification { return v.justification }

// SetJustification sets the view's centring mode.
func (v *DrawingView) SetJustification(j types.ViewJustification) { v.justification = j }

// setAlignment locks the view to anchor on the given axis, or frees it (InPosition clears the lock).
func (v *DrawingView) setAlignment(anchor string, a types.DrawingViewAlignment) {
	if a == types.InPositionViewAlignment {
		v.alignedTo, v.alignment = "", types.InPositionViewAlignment
		return
	}
	v.alignedTo, v.alignment = anchor, a
}

// Rotate sets the named view's rotation about its centre (degrees) and re-projects it (#1988).
func (vs *DrawingViews) Rotate(name string, deg float64) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no view named %q", name)
	}
	v.SetRotationDeg(deg)
	vs.Recompute()
	return nil
}

// Align locks the named view to anchor on a shared axis (or frees it with InPosition) and optionally
// sets its justification, then re-projects so the lock takes effect (#1988). It errors when the view
// or a required anchor is missing, or a view is aligned to itself.
func (vs *DrawingViews) Align(name, anchor string, a types.DrawingViewAlignment, j *types.ViewJustification) error {
	v, ok := vs.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no view named %q to align", name)
	}
	if a != types.InPositionViewAlignment {
		if anchor == name {
			return fmt.Errorf("drawing: a view cannot be aligned to itself (%q)", name)
		}
		if _, ok := vs.ByName(anchor); !ok {
			return fmt.Errorf("drawing: no anchor view named %q to align %q to", anchor, name)
		}
	}
	v.setAlignment(anchor, a)
	if j != nil {
		v.SetJustification(*j)
	}
	vs.Recompute()
	return nil
}

// applyAlignments pulls every locked view's cross-axis centre onto its anchor's, so moving an anchor
// drags its aligned views. Horizontal alignment shares the anchor's Y, vertical shares its X. It runs
// before the projection pass; a lock whose anchor is missing is ignored.
func (vs *DrawingViews) applyAlignments() {
	for _, v := range vs.items {
		if !v.IsAligned() {
			continue
		}
		anchor, ok := vs.ByName(v.alignedTo)
		if !ok {
			continue
		}
		switch v.alignment {
		case types.HorizontalViewAlignment:
			v.centerY = anchor.centerY
		case types.VerticalViewAlignment:
			v.centerX = anchor.centerX
		}
	}
}
