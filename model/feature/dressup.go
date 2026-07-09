// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Dress-up features operate on existing topology picked by the user — edges to
// round/chamfer, faces to shell/draft — held as reference keys (topo lineage keys,
// M07) and re-resolved against the running body each recompute via the topological-
// naming rebind. A picked edge "the same" after an upstream edit re-resolves; one
// that genuinely vanished makes the feature go Sick and surface for re-selection
// (parametric-cad §7, ADR-0010). The rounding/cut geometry itself is kernel phase B
// (rolling-ball fillets), so once inputs resolve these features report
// [ErrDeferred] (→ health.Warning) and pass the body through unchanged.

// FilletEdgeSet is one edge set of a fillet definition (the reference's
// FilletConstantRadiusEdgeSet / FilletVariableRadiusEdgeSet): a constant Radius over the
// whole set, or — when Radius is nil — a variable StartRadius→EndRadius over a single edge
// (radius runs linearly from the edge's start vertex to its end vertex). RadiusPoints are
// optional intermediate (position, radius) stops along that variable edge (#695), each at a
// fraction 0<T<1 of the edge, strictly increasing in T — the reference's FilletRadius set points.
type FilletEdgeSet struct {
	EdgeKeys     [][]byte
	Radius       func() float64
	StartRadius  func() float64
	EndRadius    func() float64
	RadiusPoints []FilletRadiusPoint
}

// FilletRadiusPoint is one intermediate radius stop on a variable fillet edge: T is the fraction
// along the edge (start vertex = 0, end vertex = 1), Radius the rolling-ball radius there.
type FilletRadiusPoint struct {
	T      float64
	Radius func() float64
}

// variable reports whether the set carries a start→end radius instead of a constant one.
func (s FilletEdgeSet) variable() bool { return s.Radius == nil }

// FilletCornerType aliases the public corner-treatment discriminator (ADR-0018).
type FilletCornerType = types.FilletCornerType

// FilletCrossSection aliases the public blend cross-section shape (M36-F08, ADR-0018): arc (G1,
// default), G2 (curvature-continuous), or conic (rho-controlled).
type FilletCrossSection = types.FilletCrossSection

// Fillet cross-section shapes (aliases of the canonical api/types values).
const (
	FilletArc   = types.FilletSectionArc
	FilletG2    = types.FilletSectionG2
	FilletConic = types.FilletSectionConic
)

// FilletDefinition rounds selected edges. EdgeKeys+Radius is the original single
// constant-radius form; EdgeSets (when non-empty) takes precedence and carries any mix of
// constant and variable sets (#323). CornerType selects how a vertex where two filleted edges
// meet (third edge sharp) is treated — miter crease (default/zero), or a full round.
type FilletDefinition struct {
	EdgeKeys        [][]byte
	Radius          func() float64
	EdgeSets        []FilletEdgeSet
	CornerType      FilletCornerType
	ConcaveStrategy types.FilletConcaveStrategy // concave edges: outward fill (zero/default) or inward recess
	// CrossSection selects the blend cross-section shape (M36-F08): the default arc (G1), a
	// curvature-continuous G2, or a rho-controlled conic. Rho sets a conic's fullness (0<ρ<1,
	// 0.5=parabola). Non-arc sections build via the swept ruling band (no analytic cylinder).
	CrossSection FilletCrossSection
	Rho          float64
	// GeomEdges are edges selected by a serialized GEOMETRIC descriptor rather than an
	// Oblikovati lineage key — the path an external author (the NX exporter, M8/ADR-0040)
	// uses, since it cannot synthesize lineage. They bind to the running body's edges at
	// recompute (see bindGeomEdges) and fold into the edge list; absent for a normal
	// Oblikovati-authored fillet.
	GeomEdges []topo.GeometricEdgeRef
	// EdgeAnchors maps an EdgeKeys entry (raw reference key, as a string) to the edge's
	// midpoint captured when the user picked it. It feeds the GEOMETRIC recovery tier
	// (ADR-0043 P6b): when a lost key's parent has several surviving siblings, the anchor
	// disambiguates by nearness. Absent for an older recipe or an edit-mode retained key —
	// such a reference degrades to exact/ancestral recovery only.
	EdgeAnchors map[string]math.Point3
}

// FilletType reports the definition's discriminator: always an edge fillet for now (the
// reference's face and full-round fillets are follow-ups tracked on #323).
func (d *FilletDefinition) FilletType() types.FilletType { return types.EdgeFillet }

// FilletFeature is an edge fillet over one or more constant/variable radius edge sets.
type FilletFeature struct{ def *FilletDefinition }

// Definition returns the fillet recipe.
func (f *FilletFeature) Definition() *FilletDefinition { return f.def }

// Kind implements [Feature].
func (f *FilletFeature) Kind() string { return "fillet" }

// Recompute rounds the picked convex edges on the running body with a real rolling-ball
// blend (cylinder faces; planar ruling strips for variable sets). See fillet.go.
func (f *FilletFeature) Recompute(in Input) (Output, error) {
	prof := blendProfile{cross: f.def.CrossSection, rho: f.def.Rho}
	if len(f.def.EdgeSets) > 0 {
		return filletBodySets(in, f.def.EdgeSets, f.def.CornerType, f.def.ConcaveStrategy, prof, "fillet")
	}
	keys, err := bindGeomEdges(in, f.def.EdgeKeys, f.def.GeomEdges, "fillet")
	if err != nil {
		return Output{}, err
	}
	return filletBody(in, keys, callOrZero(f.def.Radius), f.def.CornerType, f.def.ConcaveStrategy, prof, "fillet", f.def.EdgeAnchors)
}

// ChamferType aliases the public chamfer-mode discriminator (ADR-0018).
type ChamferType = types.ChamferType

// ChamferConcaveStrategy aliases the public concave-edge strategy discriminator (ADR-0018).
type ChamferConcaveStrategy = types.ChamferConcaveStrategy

// ChamferDefinition bevels selected edges. Type (M20-F03) selects the setback mode: equal
// Distance on both faces (the default / zero value), two independent distances (Distance +
// Distance2, asymmetric), or a Distance plus the chamfer-face Angle. FlatCorners blends a
// vertex where three selected edges meet into a flat triangular face — only for the
// equal-distance mode; an asymmetric chamfer leaves the corner planes to meet at a point.
// ConcaveStrategy applies only to CONCAVE (internal) edges: fill the inside corner with material
// (outward, the zero-value default) or cut a recessed relief groove (inward).
type ChamferDefinition struct {
	EdgeKeys        [][]byte
	Distance        func() float64
	Distance2       func() float64 // twoDistances: setback on the second face
	Angle           func() float64 // distanceAndAngle: chamfer-face angle (radians)
	Type            ChamferType    // zero value ⇒ equal-distance
	FlatCorners     bool
	ConcaveStrategy ChamferConcaveStrategy  // zero value ⇒ outward (fill the inside corner)
	GeomEdges       []topo.GeometricEdgeRef // externally-authored edges by geometric descriptor (see FilletDefinition.GeomEdges)
	// EdgeAnchors maps an EdgeKeys entry to its mint-time midpoint for the geometric recovery
	// tier (ADR-0043 P6b); see FilletDefinition.EdgeAnchors.
	EdgeAnchors map[string]math.Point3
}

// ChamferFeature bevels selected edges (equal-distance, two-distance, or distance-and-angle).
type ChamferFeature struct {
	def      *ChamferDefinition
	featName string
}

func (c *ChamferFeature) Definition() *ChamferDefinition { return c.def }
func (c *ChamferFeature) Kind() string                   { return "chamfer" }

// Recompute bevels each selected (convex) edge by cutting a wedge tool along it via the
// boolean; the two setbacks come from the mode (see chamferSetbacks). Flat-corner blending
// applies to every mode — the blend is built from the per-face setbacks, so an asymmetric
// (two-distance / distance-angle) corner blends just like a symmetric one. See chamfer.go.
func (c *ChamferFeature) Recompute(in Input) (Output, error) {
	d1, d2, err := chamferSetbacks(c.def)
	if err != nil {
		return Output{}, err
	}
	keys, err := bindGeomEdges(in, c.def.EdgeKeys, c.def.GeomEdges, "chamfer")
	if err != nil {
		return Output{}, err
	}
	return chamferEdges(in, keys, d1, d2, featOr(c.featName, "chamfer"), c.def.FlatCorners, c.def.ConcaveStrategy, c.def.EdgeAnchors)
}

// ShellDefinition hollows a body, removing the selected faces, to a wall thickness.
type ShellDefinition struct {
	RemovedFaceKeys [][]byte
	Thickness       func() float64
	GeomFaces       []topo.GeometricFaceRef // externally-authored removed faces by geometric descriptor (ADR-0040)
	Direction       ops.ShellDirection      // which side the wall grows onto (default ShellInside) — #1864
}

// ShellFeature hollows a solid.
type ShellFeature struct {
	def      *ShellDefinition
	featName string
}

func (s *ShellFeature) Definition() *ShellDefinition { return s.def }
func (s *ShellFeature) Kind() string                 { return "shell" }

// Recompute hollows the running body to the wall thickness, opening the removed faces. See
// shell.go.
func (s *ShellFeature) Recompute(in Input) (Output, error) {
	keys, err := bindGeomFaces(in, s.def.RemovedFaceKeys, s.def.GeomFaces, featOr(s.featName, "shell"))
	if err != nil {
		return Output{}, err
	}
	return shellBody(in, keys, callOrZero(s.def.Thickness), s.def.Direction, featOr(s.featName, "shell"))
}

// FaceDraftDefinition tapers selected faces by an angle about a pull direction. Neutral, when set, is
// the fixed parting plane each face pivots on (#1801); nil ⇒ the implicit lowest-vertex hinge.
type FaceDraftDefinition struct {
	FaceKeys  [][]byte
	PullDir   math.Vector3
	Neutral   *geom.Plane
	Angle     func() float64
	GeomFaces []topo.GeometricFaceRef // externally-authored drafted faces by geometric descriptor (ADR-0040)
}

// FaceDraftFeature applies draft to faces.
type FaceDraftFeature struct{ def *FaceDraftDefinition }

func (d *FaceDraftFeature) Definition() *FaceDraftDefinition { return d.def }
func (d *FaceDraftFeature) Kind() string                     { return "draft" }

// Recompute tapers the picked faces about the pull direction by the angle (see draft.go).
func (d *FaceDraftFeature) Recompute(in Input) (Output, error) {
	keys, err := bindGeomFaces(in, d.def.FaceKeys, d.def.GeomFaces, "draft")
	if err != nil {
		return Output{}, err
	}
	return draftBody(in, keys, d.def.PullDir, d.def.Neutral, callOrZero(d.def.Angle), "draft")
}

// ThreadDefinition applies thread data to a cylindrical face. Cut=false is a cosmetic thread
// (data + display, solid unchanged); Cut=true models a real thread (a helical groove cut).
// Class, Tapered, and ModelDiameter are the #325 parity fields: the tolerance class recorded
// on the spec, the pipe-thread flag (the reference's TaperedThreadInfo split — data-only, a
// cut tapered thread needs a conical face and errors), and which thread diameter the modeled
// cylindrical face represents (zero value = major, the common case).
type ThreadDefinition struct {
	FaceKey       []byte
	Designation   string
	Cut           bool
	Class         string
	Tapered       bool
	ModelDiameter types.ModelDiameterFromThread
	// Offset and Length limit the thread to a sub-span of the face along its axis (Inventor's
	// ThreadOffset / ThreadDepth): the thread runs from the face's min axial extent + Offset for
	// Length (cm). A nil Offset ⇒ 0; a nil or zero Length ⇒ the full face (Inventor's FullDepth).
	// A double-ended stud threads only its two ends by giving each thread its own Offset+Length.
	Offset func() float64
	Length func() float64
	// FaceAnchors maps FaceKey to its mint-time centroid for the geometric recovery tier
	// (ADR-0043 P6 / #1579); see FilletDefinition.EdgeAnchors.
	FaceAnchors map[string]math.Point3
}

// ThreadFeature tags a cylindrical face with a cosmetic thread (Inventor's ThreadFeature): it
// records the resolved thread data and leaves the solid unchanged. Cut-thread geometry (a real
// helical groove) is a separate modeled feature; the cosmetic thread is the data + display.
type ThreadFeature struct {
	def  *ThreadDefinition
	spec *ThreadSpec // resolved on the last recompute (nil until then)
}

func (t *ThreadFeature) Definition() *ThreadDefinition { return t.def }
func (t *ThreadFeature) Kind() string                  { return "thread" }

// Spec returns the thread data resolved on the last recompute (nil if it never bound).
func (t *ThreadFeature) Spec() *ThreadSpec { return t.spec }

// Recompute parses the designation, binds the cylindrical face, records the thread spec, and
// passes the (unchanged) solid through. A bad designation, a lost face, or a non-cylindrical
// face makes the feature Sick — as does cutting a tapered (pipe) thread, which would need a
// conical face the feature doesn't model yet.
func (t *ThreadFeature) Recompute(in Input) (Output, error) {
	spec, err := ParseThreadDesignation(t.def.Designation)
	if err != nil {
		return Output{}, err
	}
	spec.Class, spec.Tapered = t.def.Class, t.def.Tapered
	spec.ModelDiameter = t.def.ModelDiameter
	if spec.ModelDiameter == 0 {
		spec.ModelDiameter = types.ThreadMajorDiameter
	}
	if t.def.Tapered && t.def.Cut {
		return Output{}, fmt.Errorf("thread %q: a cut tapered (pipe) thread needs a conical face; model it cosmetic", t.def.Designation)
	}
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	face, mt, err := bindFace(body, t.def.FaceKey, anchorFor(t.def.FaceKey, t.def.FaceAnchors))
	if err != nil {
		return Output{}, fmt.Errorf("thread: %w", err)
	}
	cyl, ok := face.Geometry().(geom.Cylinder)
	if !ok {
		return Output{}, fmt.Errorf("thread %q: face is not cylindrical (%T)", t.def.Designation, face.Geometry())
	}
	vFaceMin, vFaceMax := axialExtent(face.RangeBox(), cyl)
	vMin, vMax := resolveThreadSpan(vFaceMin, vFaceMax, t.def.Offset, t.def.Length)
	spec.Internal = bodyHasMaterialOutside(body, cyl, (vMin+vMax)/2, (spec.MajorDiameter-spec.MinorDiameter)/2/10)
	t.spec = &spec
	if !t.def.Cut {
		return Output{Bodies: in.Bodies, Heals: faceHeal(t.def.FaceKey, mt)}, nil // cosmetic: solid unchanged
	}
	// Modeled (cut) thread: retype the cylindrical face to a threaded surface — O(1), no
	// boolean — so it tessellates and measures as real threaded geometry. The span honours the
	// thread's offset/length so a partial cut thread grooves only its run.
	threaded := geom.ThreadedCylinder{
		Cylinder: cyl, Pitch: spec.Pitch / 10, Depth: (spec.MajorDiameter - spec.MinorDiameter) / 2 / 10,
		Internal: spec.Internal, RightHanded: spec.RightHanded, VMin: vMin, VMax: vMax,
	}
	out := make([]*topo.Body, len(in.Bodies))
	copy(out, in.Bodies)
	// Target the RESOLVED face's current key, not the stored one: a healed thread bound to a
	// recovered sibling whose live key differs from t.def.FaceKey (ADR-0043 P6, mirrors edges).
	threadedBody, err := ops.ReplaceFaceSurface(body, face.ReferenceKey(), threaded)
	if err != nil {
		return Output{}, err
	}
	out[len(out)-1] = threadedBody // runningBody is the last body
	return Output{Bodies: out, Heals: faceHeal(t.def.FaceKey, mt)}, nil
}

// resolveFacesThenDefer resolves face keys against the running body and, if all bind, defers
// the geometry (passthrough + ErrDeferred); a lost key is a Sick error. (Still used by the
// thread cosmetic feature.)
func resolveFacesThenDefer(in Input, keys [][]byte, kind string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	// Recover lost-but-ancestral faces so a deferred feature is not falsely Sick; ErrDeferred
	// already classifies the deferral as a Warning, so the heals need no separate surfacing.
	if _, _, err := resolveFaces(body, keys, nil); err != nil {
		return Output{}, fmt.Errorf("%s: %w", kind, err)
	}
	return Output{Bodies: in.Bodies}, ErrDeferred
}

// runningBody returns the body a dress-up feature operates on (the last running
// body), erroring if there is none.
func runningBody(in Input) (*topo.Body, error) {
	if len(in.Bodies) == 0 {
		return nil, errors.New("no body to operate on")
	}
	return in.Bodies[len(in.Bodies)-1], nil
}

// DressUpFeatures adds dress-up features into the engine.
type DressUpFeatures struct{ engine *PartFeatures }

// NewDressUpFeatures binds the collection to an engine.
func NewDressUpFeatures(engine *PartFeatures) *DressUpFeatures { return &DressUpFeatures{engine} }

// AddFillet rounds the given edges (by reference key) to radius, mitering shared corners.
func (c *DressUpFeatures) AddFillet(edgeKeys [][]byte, radius func() float64) *PartFeature {
	return c.AddFilletCorner(edgeKeys, radius, types.FilletCornerMiter)
}

// AddFilletCorner rounds the given edges to radius with an explicit shared-corner treatment.
// Concave edges fill outward (the default); use [AddFilletConcave] to round a recess inward.
func (c *DressUpFeatures) AddFilletCorner(edgeKeys [][]byte, radius func() float64, corner FilletCornerType) *PartFeature {
	return c.authorEdgeFillet(&FilletDefinition{EdgeKeys: edgeKeys, Radius: radius, CornerType: corner})
}

// authorEdgeFillet captures mint-time edge anchors (ADR-0043 P6b) against the running body, then
// registers an edge-key fillet. Every PUBLIC edge-key fillet builder funnels through it so each
// authoring path records the geometric-recovery witness; the recipe restore uses addFillet, which
// preserves the persisted anchors and never recaptures.
func (c *DressUpFeatures) authorEdgeFillet(def *FilletDefinition) *PartFeature {
	if len(def.EdgeAnchors) == 0 {
		def.EdgeAnchors = captureEdgeAnchors(c.tipBody(), def.EdgeKeys)
	}
	return c.engine.Add(&FilletFeature{def: def})
}

// addFillet adds a fillet from a fully-built definition without capturing anchors (used by the
// recipe restore to carry fields the public builders don't take, e.g. geometric edge refs, and
// the persisted anchors themselves).
func (c *DressUpFeatures) addFillet(def *FilletDefinition) *PartFeature {
	return c.engine.Add(&FilletFeature{def: def})
}

// AddFilletCross rounds the given edges to radius with a chosen cross-section shape (M36-F08): arc
// (G1, the default), G2 (curvature-continuous), or conic with fullness rho. Shared corners miter.
func (c *DressUpFeatures) AddFilletCross(edgeKeys [][]byte, radius func() float64, cross FilletCrossSection, rho float64) *PartFeature {
	return c.authorEdgeFillet(&FilletDefinition{
		EdgeKeys: edgeKeys, Radius: radius, CornerType: types.FilletCornerMiter, CrossSection: cross, Rho: rho,
	})
}

// AddFilletConcave rounds the given edges to radius with an explicit concave-edge strategy: outward
// fills the inside corner with material (the default), inward rounds a recess into it. Convex edges
// are unaffected by the strategy. Shared corners miter.
func (c *DressUpFeatures) AddFilletConcave(edgeKeys [][]byte, radius func() float64, concave types.FilletConcaveStrategy) *PartFeature {
	return c.authorEdgeFillet(&FilletDefinition{EdgeKeys: edgeKeys, Radius: radius, CornerType: types.FilletCornerMiter, ConcaveStrategy: concave})
}

// AddFilletSets rounds any mix of constant and variable radius edge sets in one feature
// (the reference's FilletDefinition edge-set model, #323), mitering shared corners.
func (c *DressUpFeatures) AddFilletSets(sets []FilletEdgeSet) *PartFeature {
	return c.AddFilletSetsCorner(sets, types.FilletCornerMiter)
}

// AddFilletSetsCorner rounds the edge sets with an explicit shared-corner treatment.
func (c *DressUpFeatures) AddFilletSetsCorner(sets []FilletEdgeSet, corner FilletCornerType) *PartFeature {
	return c.engine.Add(&FilletFeature{def: &FilletDefinition{EdgeSets: sets, CornerType: corner}})
}

// AddFaceFillet rounds the edges shared between two face sets with a constant-radius rolling-ball
// blend (#694, the adjacent-faces case of FilletConstantRadiusFaceSet).
func (c *DressUpFeatures) AddFaceFillet(faceKeysA, faceKeysB [][]byte, radius func() float64) *PartFeature {
	return c.engine.Add(&FaceFilletFeature{def: &FaceFilletDefinition{FaceKeysA: faceKeysA, FaceKeysB: faceKeysB, Radius: radius}})
}

// AddRuleFillet rounds the running body's edges that match a dihedral rule, all at one radius
// (#486, the plastic-part rule fillet).
func (c *DressUpFeatures) AddRuleFillet(rule RuleFilletRule, radius func() float64) *PartFeature {
	return c.engine.Add(&RuleFilletFeature{def: &RuleFilletDefinition{Rule: rule, Radius: radius}})
}

// AddChamfer bevels the given edges by distance, blending three-edge corners flat (the
// default treatment). Use [AddChamferCorners] to choose the pointy corner instead.
func (c *DressUpFeatures) AddChamfer(edgeKeys [][]byte, distance func() float64) *PartFeature {
	return c.AddChamferCorners(edgeKeys, distance, true)
}

// AddChamferCorners bevels the given edges by distance; flatCorners selects whether a
// three-edge corner is blended into a flat triangular face (true) or left pointy (false).
// Concave edges fill outward (the default); use [AddChamferConcave] to relieve them inward.
func (c *DressUpFeatures) AddChamferCorners(edgeKeys [][]byte, distance func() float64, flatCorners bool) *PartFeature {
	return c.authorChamfer(&ChamferDefinition{EdgeKeys: edgeKeys, Distance: distance, Type: types.ChamferDistance, FlatCorners: flatCorners})
}

// AddChamferConcave bevels the given edges by distance with an explicit concave-edge strategy:
// outward fills the inside corner with material (the default), inward cuts a recessed relief
// groove. Convex edges are unaffected by the strategy. Three-edge corners blend flat.
func (c *DressUpFeatures) AddChamferConcave(edgeKeys [][]byte, distance func() float64, flatCorners bool, strategy ChamferConcaveStrategy) *PartFeature {
	return c.authorChamfer(&ChamferDefinition{EdgeKeys: edgeKeys, Distance: distance, Type: types.ChamferDistance, FlatCorners: flatCorners, ConcaveStrategy: strategy})
}

// AddChamferTwoDistances bevels the given edges with independent setbacks on the two adjacent
// faces (an asymmetric chamfer, M20-F03). Three-edge corners blend flat by default, like the
// equal-distance mode.
func (c *DressUpFeatures) AddChamferTwoDistances(edgeKeys [][]byte, distance, distance2 func() float64) *PartFeature {
	return c.authorChamfer(&ChamferDefinition{EdgeKeys: edgeKeys, Distance: distance, Distance2: distance2, Type: types.ChamferTwoDistances, FlatCorners: true})
}

// AddChamferDistanceAngle bevels the given edges by a setback on the first face and the
// chamfer-face angle (radians), M20-F03. Three-edge corners blend flat by default, like the
// equal-distance mode.
func (c *DressUpFeatures) AddChamferDistanceAngle(edgeKeys [][]byte, distance, angle func() float64) *PartFeature {
	return c.authorChamfer(&ChamferDefinition{EdgeKeys: edgeKeys, Distance: distance, Angle: angle, Type: types.ChamferDistanceAndAngle, FlatCorners: true})
}

// authorChamfer captures mint-time edge anchors (ADR-0043 P6b) against the running body, then
// registers the chamfer. Every PUBLIC chamfer builder funnels through it so each authoring path
// (GUI, wire API, assembly, programmatic) records the geometric-recovery witness; the recipe
// restore calls addChamfer directly so reopening a document never recaptures or rewrites anchors.
func (c *DressUpFeatures) authorChamfer(def *ChamferDefinition) *PartFeature {
	if len(def.EdgeAnchors) == 0 {
		def.EdgeAnchors = captureEdgeAnchors(c.tipBody(), def.EdgeKeys)
	}
	return c.addChamfer(def)
}

// addChamfer registers a chamfer feature with the given definition (no anchor capture — shared
// by the recipe restore, which carries persisted anchors of its own).
func (c *DressUpFeatures) addChamfer(def *ChamferDefinition) *PartFeature {
	cf := &ChamferFeature{def: def}
	pf := c.engine.Add(cf)
	pf.SetName(c.engine.UniqueName("Chamfer"))
	cf.featName = pf.name
	return pf
}

// AddShell hollows the body, removing the given faces, to thickness.
func (c *DressUpFeatures) AddShell(removedFaceKeys [][]byte, thickness func() float64) *PartFeature {
	sf := &ShellFeature{def: &ShellDefinition{RemovedFaceKeys: removedFaceKeys, Thickness: thickness}}
	pf := c.engine.Add(sf)
	pf.SetName(c.engine.UniqueName("Shell"))
	sf.featName = pf.name
	return pf
}

// AddDraft tapers the given faces by angle about the default +Z pull direction.
func (c *DressUpFeatures) AddDraft(faceKeys [][]byte, angle func() float64) *PartFeature {
	return c.AddDraftPull(faceKeys, math.V3(0, 0, 1), angle)
}

// AddDraftPull tapers the given faces by angle about an explicit pull direction (implicit hinge).
func (c *DressUpFeatures) AddDraftPull(faceKeys [][]byte, pull math.Vector3, angle func() float64) *PartFeature {
	return c.AddDraftPullNeutral(faceKeys, pull, nil, angle)
}

// AddDraftPullNeutral tapers the given faces by angle about an explicit pull direction, pivoting each
// face on the fixed neutral (parting) plane when one is given (#1801); nil ⇒ the implicit hinge.
func (c *DressUpFeatures) AddDraftPullNeutral(faceKeys [][]byte, pull math.Vector3, neutral *geom.Plane, angle func() float64) *PartFeature {
	return c.engine.Add(&FaceDraftFeature{def: &FaceDraftDefinition{FaceKeys: faceKeys, PullDir: pull, Neutral: neutral, Angle: angle}})
}

// addShell / addFaceDraft add a face dress-up from a fully-built definition (used by the
// recipe restore to carry geometric face refs the public builders don't take).
func (c *DressUpFeatures) addShell(def *ShellDefinition) *PartFeature {
	sf := &ShellFeature{def: def}
	pf := c.engine.Add(sf)
	pf.SetName(c.engine.UniqueName("Shell"))
	sf.featName = pf.name
	return pf
}

func (c *DressUpFeatures) addFaceDraft(def *FaceDraftDefinition) *PartFeature {
	return c.engine.Add(&FaceDraftFeature{def: def})
}

// AddThread tags a cylindrical face with thread data; cut=true models a real (cut) thread,
// cut=false a cosmetic one.
func (c *DressUpFeatures) AddThread(faceKey []byte, designation string, cut bool) *PartFeature {
	return c.AddThreadDef(&ThreadDefinition{FaceKey: faceKey, Designation: designation, Cut: cut})
}

// AddThreadDef adds a thread from a full definition (class / tapered / model diameter, #325). It
// captures the threaded face's mint-time anchor against the running body for the geometric
// recovery tier (ADR-0043 P6 / #1579); every authoring path funnels here, while the recipe restore
// uses addThreadDef so reopening a document never recaptures or rewrites anchors.
func (c *DressUpFeatures) AddThreadDef(def *ThreadDefinition) *PartFeature {
	if len(def.FaceAnchors) == 0 {
		def.FaceAnchors = captureFaceAnchors(c.tipBody(), [][]byte{def.FaceKey})
	}
	return c.addThreadDef(def)
}

// addThreadDef registers a thread from a fully-built definition without capturing anchors (the
// recipe restore path, which carries the persisted anchors of its own).
func (c *DressUpFeatures) addThreadDef(def *ThreadDefinition) *PartFeature {
	return c.engine.Add(&ThreadFeature{def: def})
}
