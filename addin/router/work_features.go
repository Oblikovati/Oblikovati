// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// setWorkFeatureVisible shows or hides the datum work plane, axis, or point named by the request's
// ref — the post-create visibility toggle Inventor exposes on every work feature (#1856). It
// resolves the ref against the active model's work geometry (origin frame + user datums) and
// recomputes so subscribers see the change; an unknown ref is a clean error.
func setWorkFeatureVisible(_ *app.Session, host workHost, in wire.SetWorkFeatureVisibleArgs) (wire.OKResult, error) {
	g := host.WorkGeometry()
	ref := feature.ParseWorkRef(in.Ref)
	switch {
	case setPlaneVisible(g, ref, in.Visible),
		setAxisVisible(g, ref, in.Visible),
		setPointVisible(g, ref, in.Visible):
		host.Recompute()
		return wire.OKResult{OK: true}, nil
	default:
		return wire.OKResult{}, fmt.Errorf("workFeatures.setVisible: no work plane, axis, or point named %q", in.Ref)
	}
}

// setPlaneVisible sets a datum plane's visibility, reporting whether the ref named one.
func setPlaneVisible(g *feature.WorkGeometry, ref feature.WorkRef, visible bool) bool {
	wp, err := g.WorkPlaneByRef(ref)
	if err != nil {
		return false
	}
	wp.SetVisible(visible)
	return true
}

// setAxisVisible sets a datum axis's visibility, reporting whether the ref named one.
func setAxisVisible(g *feature.WorkGeometry, ref feature.WorkRef, visible bool) bool {
	wa, ok := g.AxisByRef(ref)
	if ok {
		wa.SetVisible(visible)
	}
	return ok
}

// setPointVisible sets a datum point's visibility, reporting whether the ref named one.
func setPointVisible(g *feature.WorkGeometry, ref feature.WorkRef, visible bool) bool {
	wp, ok := g.WorkPointByRef(ref)
	if ok {
		wp.SetVisible(visible)
	}
	return ok
}
