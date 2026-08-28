// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// ExtrudeDefinition is the recipe for an extrude (the Definition of the triangle):
// a sketch profile, the operation against existing material, the extent, and a
// taper. It re-derives the profile from its sketch each recompute (so sketch edits
// flow through), going sick if the profile is gone or open.
type ExtrudeDefinition struct {
	Sketch         *sketch.Sketch
	ProfileIndices []int // one or more sketch regions, extruded together into one feature
	// ProfileSeeds are interior seed points (sketch 2-D, cm) selecting the regions by
	// containment. When present they are resolved to region indices EVERY recompute, so an
	// externally-authored selection survives the sketch being re-solved between load and
	// recompute (which reorders the DCEL regions and would otherwise strand ProfileIndices on
	// the wrong cells — #region-seed). Empty ⇒ ProfileIndices is used directly.
	ProfileSeeds [][]float64
	Operation    ops.PartFeatureOperation
	Extent       Extent
	Taper        float64 // draft angle (radians); 0 in phase A (planar sides)
	// NoDissolve keeps abutting profiles as separate prisms instead of fusing them into one (#38).
	// The dissolve is on by default (it fixes the coincident-wall crack a slot-with-corner-reliefs cut
	// leaves); a caller that finds the merged tool trips a downstream boolean fragility on a particular
	// part sets this to fall back to the per-region prisms for that part.
	NoDissolve bool
}

// ExtrudeFeature turns a profile into a prism and combines it with the running
// body state. It is the reference sketched feature (PBI-092).
type ExtrudeFeature struct {
	def      *ExtrudeDefinition
	featName string
	tool     *topo.Body // last prism built, exposed so a pattern can replicate this feature
}

// ToolBody returns the prism this feature last combined into the model — the clean tool a
// pattern replicates at each occurrence (more robust than diffing before/after bodies,
// especially for curved geometry). It is nil before the first recompute.
func (e *ExtrudeFeature) ToolBody() *topo.Body { return e.tool }

// Definition returns the extrude recipe (round-trippable, serializable).
func (e *ExtrudeFeature) Definition() *ExtrudeDefinition { return e.def }

// Kind implements [Feature].
func (e *ExtrudeFeature) Kind() string { return "extrude" }

// DistanceValue returns the current extent distance (database units) — the value a
// feature editor shows when re-opening the extrude.
func (e *ExtrudeFeature) DistanceValue() float64 { return e.def.Extent.distance() }

// SetDistance replaces the extent with a constant distance, keeping the extent type and
// direction — the feature editor's distance field writes through here. Mark the feature
// dirty and recompute afterwards for the change to take effect.
func (e *ExtrudeFeature) SetDistance(d float64) {
	e.def.Extent.Distance = func() float64 { return d }
}

// Operation returns the boolean operation applied against the existing bodies.
func (e *ExtrudeFeature) Operation() ops.PartFeatureOperation { return e.def.Operation }

// SetOperation changes the boolean operation (join/cut/intersect/new-body).
func (e *ExtrudeFeature) SetOperation(op ops.PartFeatureOperation) { e.def.Operation = op }

// Extent returns the feature's termination (type/direction/distances/targets); SetExtent
// replaces it — the feature editor and Extrude tool write the chosen mode through here.
func (e *ExtrudeFeature) Extent() Extent           { return e.def.Extent }
func (e *ExtrudeFeature) SetExtent(ext Extent)     { e.def.Extent = ext }
func (e *ExtrudeFeature) Taper() float64           { return e.def.Taper }
func (e *ExtrudeFeature) SetTaper(radians float64) { e.def.Taper = radians }

// Recompute resolves the profile, computes the extent span (distance/through-all/to-face/
// …), builds the prism solid the span sweeps, and applies the operation against the
// running bodies.
func (e *ExtrudeFeature) Recompute(in Input) (Output, error) {
	profiles, err := e.resolveProfiles()
	if err != nil {
		return Output{}, err
	}
	plane := e.def.Sketch.Plane()
	sp, err := e.resolveSpan(in.Bodies, plane, outerPolygons(profiles))
	if err != nil {
		return Output{}, err
	}
	if sp.depth() == 0 {
		return Output{}, errors.New("extrude: the extent has zero depth")
	}
	if e.def.Operation == ops.Surface {
		return e.recomputeSurface(in, profiles, plane, sp)
	}
	prisms := profilePrismsDissolving(profiles, plane, sp, e.def.Taper, e.featName, in.Diag, !e.def.NoDissolve)
	e.tool = mergePrisms(prisms, e.featName) // a pattern replicates the whole tool, lumps and all
	bodies, err := combinePrisms(in, prisms, e.tool, e.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// recomputeSurface builds an open (unmerged) sheet tool and applies it — the Surface operation keeps
// the profile as a surface body rather than sweeping a solid prism.
func (e *ExtrudeFeature) recomputeSurface(in Input, profiles []*sketch.Profile, plane sketch.Plane, sp span) (Output, error) {
	e.tool = e.buildTool(profiles, plane, sp, in.Diag)
	bodies, err := combine(in, e.tool, e.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// buildTool extrudes each selected region into a prism over the span and merges the
// prisms into one body. The regions are distinct cells of the same sketch, so they never
// overlap — a shell merge is exactly their union and avoids the intersecting Join.
func (e *ExtrudeFeature) buildTool(profiles []*sketch.Profile, plane sketch.Plane, sp span, rec *diag.Recorder) *topo.Body {
	// Surface (kSurfaceOperation): extrude the profile walls only — an open, uncapped sheet
	// body — rather than a capped solid prism. #1858.
	if e.def.Operation == ops.Surface {
		return buildProfileSheets(profiles, plane, sp, e.def.Taper, e.featName, rec)
	}
	return buildProfilePrisms(profiles, plane, sp, e.def.Taper, e.featName, rec)
}

// outerPolygons returns each profile's outer-loop polygon, the input the span resolver
// (to-next ray-casting) and the prism builder both consume.
func outerPolygons(profiles []*sketch.Profile) [][]math.Point2 {
	out := make([][]math.Point2, len(profiles))
	for i, p := range profiles {
		out[i] = p.OuterLoop().Polygon()
	}
	return out
}

// resolveProfiles re-derives the selected closed regions from the sketch (the shared
// resolver), erroring (→ sick) when one is missing or open, or when none is selected. Seed
// points, when present, are resolved to indices against the CURRENT regions each recompute so
// the selection tracks a re-solved sketch (region ordering is a DCEL artifact — #region-seed).
func (e *ExtrudeFeature) resolveProfiles() ([]*sketch.Profile, error) {
	indices := e.def.ProfileIndices
	if len(e.def.ProfileSeeds) > 0 {
		indices = resolveSeeds(e.def.Sketch, e.def.ProfileSeeds, e.def.ProfileIndices)
	}
	return resolveClosedProfiles(e.def.Sketch, indices, "extrude")
}

// ExtrudeFeatures is the collection of extrude features, adding into the engine.
type ExtrudeFeatures struct {
	engine *PartFeatures
}

// NewExtrudeFeatures binds the collection to a feature engine.
func NewExtrudeFeatures(engine *PartFeatures) *ExtrudeFeatures {
	return &ExtrudeFeatures{engine: engine}
}

// AddByDistanceExtent adds an extrude of a single sketch region, growing distance (a
// closure, typically a parameter) under the given operation.
func (c *ExtrudeFeatures) AddByDistanceExtent(skt *sketch.Sketch, profileIndex int, op ops.PartFeatureOperation, distance func() float64) *PartFeature {
	return c.addByDistanceExtentProfiles(skt, []int{profileIndex}, op, distance)
}

// addByDistanceExtentProfiles adds an extrude of one or more sketch regions, merged
// into one body — the multi-region selection the Extrude tool gathers with Ctrl+click.
func (c *ExtrudeFeatures) addByDistanceExtentProfiles(skt *sketch.Sketch, profileIndices []int, op ops.PartFeatureOperation, distance func() float64) *PartFeature {
	return c.AddExtrude(skt, profileIndices, op, Extent{Type: DistanceExtent, Distance: distance}, 0)
}

// AddExtrude adds an extrude of one or more regions with a fully-specified extent
// (distance / through-all / to-face / from-to / to-next / distance-from-face), boolean
// operation, and taper (draft) angle — the general constructor the Extrude tool and the
// automation API drive. The distance constructors above delegate here.
func (c *ExtrudeFeatures) AddExtrude(skt *sketch.Sketch, profileIndices []int, op ops.PartFeatureOperation, extent Extent, taper float64) *PartFeature {
	return c.AddExtrudeFeature(&ExtrudeDefinition{
		Sketch: skt, ProfileIndices: append([]int(nil), profileIndices...),
		Operation: op, Extent: extent, Taper: taper,
	})
}

// AddExtrudeFeature registers and names an extrude built from def — the seam shared by the
// arg-based AddExtrude above and the Extrude tool, which builds the same def for both the
// live preview ([NewExtrudeFeature]) and commit so the previewed geometry is exactly what
// OK creates.
func (c *ExtrudeFeatures) AddExtrudeFeature(def *ExtrudeDefinition) *PartFeature {
	ef := NewExtrudeFeature(def)
	pf := c.engine.Add(ef)
	pf.SetName(c.engine.UniqueName("Extrusion")) // Extrusion1, Extrusion2, … (Inventor's naming)
	ef.featName = pf.name
	return pf
}

// NewExtrudeFeature builds an extrude feature value from a definition WITHOUT adding it to
// any engine — the unattached, unnamed [Feature] the live preview evaluates speculatively
// (see [PartFeatures.PreviewResult]). AddExtrudeFeature wraps this, then registers and names it.
func NewExtrudeFeature(def *ExtrudeDefinition) *ExtrudeFeature { return &ExtrudeFeature{def: def} }
