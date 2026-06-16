// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Sheet-metal bend development (M13-F04, #377). A bend develops by a non-affine point map that
// unrolls (unfold) or rolls (refold) the bend region around its neutral fibre, applied to the
// whole body by [ops.DeformBody]. Because the map moves every vertex — including those a cut
// added while flat — a slot that crosses the bend rides through it on refold, with no separate
// folded↔flat entity map. This replaces the earlier rigid-rotation approximation, which left
// the bend arc curved (so the bend looked broken and a crossing cut reached only one face).
//
// Frame (from the flange's recorded [BendPlacement]): the bend line lies on the inner surface
// (radius r from the bend-axis centre, which is the line offset by Up·r). `out` is the in-plane
// direction toward the flange; `up` is the fold normal. A point of the folded body at swept
// angle φ ∈ [0, angle] and radius ρ ∈ [r, r+t] develops to flat at across-distance φ·ρ_neutral
// from the bend line (neutral-fibre arc length) and through-thickness offset (r−ρ) along up;
// the straight flange beyond the arc continues at across ≥ angle·ρ_neutral.

// bendDevelop is the resolved frame + dimensions for developing one bend, both directions.
type bendDevelop struct {
	line                              math.Point3      // a point on the inner-edge bend line
	dir, up, out                      math.UnitVector3 // bend-line, fold-normal, in-plane-outward axes
	radius, thickness, angle, neutral float64          // inside radius, gauge, swept angle, neutral-fibre radius
}

// newBendDevelop resolves and validates the development frame from a baked transform.
func newBendDevelop(bt BendTransform) (bendDevelop, error) {
	dir, err := math.UnitVector3FromVector(bt.LineDir)
	if err != nil {
		return bendDevelop{}, fmt.Errorf("sheet-metal develop: degenerate bend line direction %v", bt.LineDir)
	}
	up, err := math.UnitVector3FromVector(bt.Up)
	if err != nil {
		return bendDevelop{}, fmt.Errorf("sheet-metal develop: degenerate fold normal %v", bt.Up)
	}
	out, err := math.UnitVector3FromVector(bt.Out)
	if err != nil {
		return bendDevelop{}, fmt.Errorf("sheet-metal develop: degenerate outward direction %v", bt.Out)
	}
	if bt.Angle <= 0 || bt.Radius <= 0 || bt.Neutral <= 0 {
		return bendDevelop{}, fmt.Errorf("sheet-metal develop: angle/radius/neutral must be positive (a=%g r=%g n=%g)", bt.Angle, bt.Radius, bt.Neutral)
	}
	return bendDevelop{line: bt.LinePoint, dir: dir, up: up, out: out, radius: bt.Radius, thickness: bt.Thickness, angle: bt.Angle, neutral: bt.Neutral}, nil
}

// lineAt returns the bend-line point sharing q's coordinate along the bend axis (so the map is
// a pure section-plane transform that preserves position along the bend line).
func (b bendDevelop) lineAt(q math.Point3) math.Point3 {
	t := b.dir.AsVector().Dot(b.line.VectorTo(q))
	return b.line.TranslateBy(b.dir.AsVector().Scale(t))
}

// radial is the unit radial direction at swept angle phi (matching the flange band's
// construction: up·(−cos φ) + out·(sin φ), so φ=0 points along −up to the inner edge).
func (b bendDevelop) radial(phi float64) math.Vector3 {
	return b.up.AsVector().Scale(-stdmath.Cos(phi)).Add(b.out.AsVector().Scale(stdmath.Sin(phi)))
}

// wall is the unit direction the straight flange runs past the arc end.
func (b bendDevelop) wall() math.Vector3 {
	return b.out.AsVector().Scale(stdmath.Cos(b.angle)).Add(b.up.AsVector().Scale(stdmath.Sin(b.angle)))
}

// foldedToFlat maps a folded-body point to its developed-flat position (unfold). Base-side
// points (swept angle ≤ 0) are unchanged.
func (b bendDevelop) foldedToFlat(q math.Point3) math.Point3 {
	o := b.lineAt(q)
	centre := o.TranslateBy(b.up.AsVector().Scale(b.radius))
	v := centre.VectorTo(q)
	av, hv := b.out.AsVector().Dot(v), b.up.AsVector().Dot(v)
	phi := stdmath.Atan2(av, -hv)
	if phi <= 0 {
		return q
	}
	var across, rho float64
	if phi < b.angle {
		rho = stdmath.Hypot(av, hv)
		across = phi * b.neutral
	} else {
		rho = b.radial(b.angle).Dot(v)               // radial offset at the arc end
		across = b.angle*b.neutral + b.wall().Dot(v) // arc length + straight run
	}
	return o.TranslateBy(b.out.AsVector().Scale(across)).TranslateBy(b.up.AsVector().Scale(b.radius - rho))
}

// flatToFolded maps a developed-flat point back onto the folded bend (refold) — the inverse of
// foldedToFlat. Base-side points (across ≤ 0) are unchanged.
func (b bendDevelop) flatToFolded(p math.Point3) math.Point3 {
	o := b.lineAt(p)
	v := o.VectorTo(p)
	across := b.out.AsVector().Dot(v)
	if across <= 0 {
		return p
	}
	rho := b.radius - b.up.AsVector().Dot(v) // through-thickness: up-offset (r−ρ) ⇒ ρ = r − up·v
	centre := o.TranslateBy(b.up.AsVector().Scale(b.radius))
	arcLen := b.angle * b.neutral
	if across <= arcLen {
		return centre.TranslateBy(b.radial(across / b.neutral).Scale(rho))
	}
	end := centre.TranslateBy(b.radial(b.angle).Scale(rho))
	return end.TranslateBy(b.wall().Scale(across - arcLen))
}
