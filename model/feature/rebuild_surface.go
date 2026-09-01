// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops/surface"
)

// The Rebuild feature (M36-F02) refits the running surface body's freeform faces to clean
// Class-A NURBS — a chosen low degree and a small, even control-point count per direction —
// so an imported or boolean-derived quilt gains the cleanliness milling and reflection
// quality depend on. It records the worst geometric deviation it incurred so the UI can show
// whether the rebuild stayed within tolerance. Like a surface offset, it acts on the most
// recent body and replaces it (kernel work in surface.RebuildFaceSurfaces).

// RebuildDefinition is the recipe for a surface rebuild: the target degree and control-point
// count in each parametric direction the body's freeform faces are refit to.
type RebuildDefinition struct {
	UDegree, VDegree int
	UCount, VCount   int
}

// RebuildFeature rebuilds the running surface body's freeform faces to clean NURBS.
type RebuildFeature struct {
	def       *RebuildDefinition
	featName  string
	deviation float64 // worst face deviation from the last recompute (a cached result, not recipe)
}

// Definition returns the rebuild recipe.
func (r *RebuildFeature) Definition() *RebuildDefinition { return r.def }

// Kind implements [Feature].
func (r *RebuildFeature) Kind() string { return "rebuild-surface" }

// Deviation returns the worst geometric deviation between the source and rebuilt faces from
// the most recent recompute (0 before the first recompute).
func (r *RebuildFeature) Deviation() float64 { return r.deviation }

// Recompute rebuilds the running surface body's freeform faces and replaces it, caching the
// achieved deviation. It errors (→ sick) when there is no target body or none of its faces is
// rebuildable.
func (r *RebuildFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "rebuild")
	if err != nil {
		return Output{}, err
	}
	if err := validateRebuild(r.def); err != nil {
		return Output{}, err
	}
	out, dev, err := surface.RebuildFaceSurfaces(target, r.def.UDegree, r.def.VDegree, r.def.UCount, r.def.VCount, 0)
	if err != nil {
		return Output{}, err
	}
	r.deviation = dev
	return Output{Bodies: replaceLast(in.Bodies, out)}, nil
}

// validateRebuild rejects a recipe whose per-direction control count cannot carry the degree.
func validateRebuild(d *RebuildDefinition) error {
	if d.UDegree < 1 || d.VDegree < 1 {
		return fmt.Errorf("rebuild: degrees %dx%d must both be >= 1", d.UDegree, d.VDegree)
	}
	if d.UCount < d.UDegree+1 || d.VCount < d.VDegree+1 {
		return fmt.Errorf("rebuild: control counts %dx%d must be >= degree+1 (%dx%d)", d.UCount, d.VCount, d.UDegree+1, d.VDegree+1)
	}
	return nil
}

// RebuildFeatures adds rebuild features into the engine.
type RebuildFeatures struct{ engine *PartFeatures }

// NewRebuildFeatures binds the collection to a feature engine.
func NewRebuildFeatures(engine *PartFeatures) *RebuildFeatures {
	return &RebuildFeatures{engine: engine}
}

// Add rebuilds the running surface body's freeform faces to a clean (uDeg×vDeg) NURBS with
// uCount×vCount control points.
func (c *RebuildFeatures) Add(uDeg, vDeg, uCount, vCount int) *PartFeature {
	def := &RebuildDefinition{UDegree: uDeg, VDegree: vDeg, UCount: uCount, VCount: vCount}
	rf := &RebuildFeature{def: def, featName: "Rebuild"}
	pf := c.engine.Add(rf)
	rf.featName = pf.name
	return pf
}
