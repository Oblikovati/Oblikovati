// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The boss: a raised cylinder on a picked face — the additive twin of the hole next door. It shares
// the drill tool builder and the placement-face resolution with [HoleFeature]; only the boolean
// differs, so the two lived in one file until the hole grew its placements and terminations
// (#1861/#1863) and pushed it past the file budget.

// BossDefinition is the recipe for a boss: a raised cylinder on a placement face.
type BossDefinition struct {
	PlacementFaceKey []byte
	Diameter         func() float64
	Height           func() float64
	// FaceAnchors maps the placement face key to its mint-time centroid for the geometric
	// recovery tier (ADR-0043 P6 / #1579); see FilletDefinition.EdgeAnchors.
	FaceAnchors map[string]math.Point3
}

// BossFeature adds a cylindrical boss to the running solid.
type BossFeature struct {
	def      *BossDefinition
	featName string
	tool     *topo.Body // the boss cylinder of the last recompute, for pattern replication
}

func (b *BossFeature) Definition() *BossDefinition { return b.def }
func (b *BossFeature) Kind() string                { return "boss" }

// Recompute resolves the placement face and raises the boss cylinder from its centroid along
// the outward normal, joining it to the running body. The tool's small entry overhang sits
// INSIDE the body (drillTool's near span), so the union always overlaps cleanly.
func (b *BossFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	face, mt, err := bindFace(body, b.def.PlacementFaceKey, anchorFor(b.def.PlacementFaceKey, b.def.FaceAnchors))
	if err != nil {
		return Output{}, fmt.Errorf("boss: %w", err)
	}
	r, h := callOrZero(b.def.Diameter)/2, callOrZero(b.def.Height)
	if r <= 0 || h <= 0 {
		return Output{}, fmt.Errorf("boss: diameter %g and height %g must be > 0", 2*r, h)
	}
	out, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0))
	if err != nil {
		return Output{}, fmt.Errorf("boss: placement face has no normal")
	}
	b.tool = drillTool(centroidOf(faceVertexPoints(face)), out, r, h, featOr(b.featName, "boss"))
	res, err := ops.BooleanWithDiagnostics(ops.Join, body, b.tool, in.Diag)
	if err != nil {
		return Output{}, fmt.Errorf("boss: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res), Heals: faceHeal(b.def.PlacementFaceKey, mt)}, nil
}

// Operation reports that a boss adds material, so a pattern of a boss unions its raised
// cylinder at each occurrence (one body with N studs) instead of copying the whole solid.
// Implements [OperationalFeature].
func (b *BossFeature) Operation() ops.PartFeatureOperation { return ops.Join }

// ToolBody returns the boss cylinder the last recompute joined, so a pattern replicates a
// clean stud at each occurrence. Implements [ToolFeature].
func (b *BossFeature) ToolBody() *topo.Body { return b.tool }
