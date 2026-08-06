// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Hole and boss features place parametric cylindrical cuts/bosses on a picked placement
// face (held by reference key, re-resolved each recompute). A hole drills at an explicit
// Center (externally-authored holes) or the face centroid, along the inward normal with an
// EXACT cylinder wall (K1b): a Through All hole on a
// planar slab via brep.CutCylindricalHole, a blind hole via brep.CutBlindCylindricalHole
// (flat bottom, or a conical drill point when PointAngle is set). A counterbore adds a flat
// recess + shoulder; a countersink a true cone recess (exact-only). Unsupported drilled/
// counterbore shapes fall back to the faceted boolean. A boss is the join-side mirror of the
// drilled hole: the same tool cylinder grown OUT of the face and unioned (#327). A lost
// placement face → Sick.

// HoleTapInfo carries thread data for a tapped hole, consumed by hole tables (M14).
type HoleTapInfo struct {
	Tapped      bool
	Designation string
	// Tapered marks a taper thread (NPT and friends) rather than a straight one — Inventor's
	// TaperedThreadInfo, a distinct tap function rather than a variant of the designation.
	Tapered bool
	// Class is the fit class the thread is cut to, e.g. "6H" (metric) or "2B" (unified).
	Class string
	// LeftHanded reverses the thread's sense. Named for the LEFT hand, not Inventor's RightHanded,
	// because a zero-valued HoleTapInfo must mean the ordinary right-hand thread — several call
	// sites build a definition as a literal, and a RightHanded bool would default them all to a
	// left-hand thread. The op layer maps Inventor's spelling onto this one.
	LeftHanded bool
}

// HoleType is the hole's SEAT: the recess (if any) the bore opens into at the placement face.
// It is orthogonal to the tap — a counterbored tapped hole is an ordinary thing — which is why the
// tap lives in its own field rather than as another member here (#1862).
type HoleType uint8

const (
	DrilledHole HoleType = iota
	CounterboreHole
	CountersinkHole
	// SpotFaceHole is a shallow facing recess: a flat-bottomed cylindrical seat that squares up a
	// cast or curved surface for a fastener head to sit on. Geometrically it is the counterbore's
	// recess — a flat-bottomed cylinder — so it is cut by the same builder; it is a distinct member
	// because the DISTINCTION is real to everything downstream (callouts, hole notes, CAM), and
	// collapsing it into a counterbore on import would lose it.
	SpotFaceHole
)

// HoleDefinition is the recipe for a hole: a placement face, diameter and depth,
// type, and optional tap data.
type HoleDefinition struct {
	PlacementFaceKey []byte
	GeomFace         *topo.GeometricFaceRef // externally-authored placement face by geometric descriptor (ADR-0040)
	// Center is an explicit drill point in model space (e.g. an exporter reading the host's
	// real hole position). Nil means drill at the placement face centroid (interactive default).
	Center          *math.Point3
	Diameter        func() float64
	Depth           func() float64
	ThroughAll      bool           // drill all the way through (depth ignored)
	CounterDiameter func() float64 // recess/sink diameter (Counterbore + Countersink)
	CounterDepth    func() float64 // counterbore recess depth (CounterboreHole)
	CounterAngle    func() float64 // countersink included angle, radians (CountersinkHole)
	PointAngle      func() float64 // drilled blind-hole point: included angle, radians (0 = flat)
	Type            HoleType
	Tap             HoleTapInfo
	// Placement is the rule locating the bores, re-resolved every recompute (#1861): a sketch's
	// centre points, an offset pair, a concentric circle, or a work point. nil ⇒ the single bore on
	// PlacementFaceKey/GeomFace at Center, which is itself the face placement.
	Placement HolePlacement
	// Termination is where the bore stops (#1863): the zero value is the plain Depth, and the
	// geometric members bottom it on ToPlane / between FromPlane and ToPlane instead. ThroughAll
	// still spells the through-everything case.
	Termination ExtentType
	ToPlane     *WorkPlane
	FromPlane   *WorkPlane
	// Clearance sizes the bore from a fastener table instead of Diameter (#1862), keeping the
	// fastener → hole link live: change the fastener and the bore follows. Zero ⇒ use Diameter.
	Clearance HoleClearanceInfo
}

// HoleFeature drills a hole into the running solid.
type HoleFeature struct {
	def      *HoleDefinition
	featName string
	tool     *topo.Body // the drill cylinder of the last recompute, for pattern replication
}

// Definition returns the hole recipe.
func (h *HoleFeature) Definition() *HoleDefinition { return h.def }

// Kind implements [Feature].
func (h *HoleFeature) Kind() string { return "hole" }

// Operation reports that a hole removes material, so a pattern of a hole re-cuts the bore at
// each occurrence (one body with N holes) instead of copying the whole solid. Implements
// [OperationalFeature].
func (h *HoleFeature) Operation() ops.PartFeatureOperation { return ops.Cut }

// ToolBody returns the drill cylinder the last recompute cut, so a pattern replicates a clean
// bore at each occurrence rather than diffing the running solid (which degenerates once the
// bore makes a face curved). For a counterbore/countersink the tool is the main bore cylinder —
// the dominant cut — so a pattern of one reproduces its bores. Implements [ToolFeature].
func (h *HoleFeature) ToolBody() *topo.Body { return h.tool }

// Recompute resolves the placement face, then drills at the explicit center (or the face
// centroid) along the inward normal (see drill), subtracting the hole from the running body.
func (h *HoleFeature) Recompute(in Input) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	sites, err := h.sites(body)
	if err != nil {
		return Output{}, err
	}
	dia, err := h.boreDiameter()
	if err != nil {
		return Output{}, err
	}
	r, depth := dia/2, callOrZero(h.def.Depth)
	if r <= 0 {
		return Output{}, fmt.Errorf("hole: diameter %g must be > 0", 2*r)
	}
	// A geometric termination measures its own depth (#1863), so only a plain-distance bore needs
	// one typed in — demanding it there too would force a meaningless number alongside the target.
	if !h.def.ThroughAll && h.def.Termination == DistanceExtent && depth <= 0 {
		return Output{}, fmt.Errorf("hole: depth %g must be > 0 (or set ThroughAll, or terminate on a face)", depth)
	}
	res, err := h.drillEvery(body, sites, r, depth, in.Diag)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, res)}, nil
}

// sites is where this hole drills: the placement's rule when one is set (#1861), else the historic
// single bore on the picked face — which is itself just the face placement, kept as its own path so
// every recipe already on disk resolves exactly as before.
func (h *HoleFeature) sites(body *topo.Body) (HoleSites, error) {
	if h.def.Placement != nil {
		return h.def.Placement.Sites(body)
	}
	face, ok := resolveHoleFace(body, h.def.faceRef(), h.def.Center)
	if !ok {
		return HoleSites{}, fmt.Errorf("hole: placement face reference lost")
	}
	into, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0).Scale(-1))
	if err != nil {
		return HoleSites{}, fmt.Errorf("hole: placement face has no normal")
	}
	return HoleSites{Centers: []math.Point3{holeCenter(h.def, face)}, Into: into}, nil
}

// faceRef is the definition's placement face, as the reference every placement resolves through.
func (d *HoleDefinition) faceRef() HoleFaceRef {
	return HoleFaceRef{Key: d.PlacementFaceKey, Geom: d.GeomFace}
}

// boreDiameter is the bore's diameter in model cm. A CLEARANCE hole takes it from the fastener
// table every recompute (#1862), so the fastener stays the thing being edited and the hole follows;
// every other hole reads the authored Diameter.
func (h *HoleFeature) boreDiameter() (float64, error) {
	if h.def.Clearance.isSet() {
		return h.def.Clearance.ClearanceDiameter()
	}
	return callOrZero(h.def.Diameter), nil
}

// drillEvery cuts one bore per site, feeding each cut's result to the next so a sketch of centre
// points comes out as one body with N holes. The FIRST bore is kept as the pattern tool: a pattern
// of a multi-hole feature replicates a representative bore, which is what its single ToolBody can
// carry (see [HoleFeature.ToolBody]).
func (h *HoleFeature) drillEvery(body *topo.Body, sites HoleSites, r, depth float64,
	rec *diag.Recorder) (*topo.Body, error) {
	for i, centre := range sites.Centers {
		bore, err := h.resolveBore(centre, sites.Into, depth)
		if err != nil {
			return nil, err
		}
		bore.entry = boreEntryOverhang(body, bore.start)
		if i == 0 {
			h.tool = h.buildTool(body, bore, sites.Into, r)
		}
		res, err := h.drill(body, bore, sites.Into, r, rec)
		if err != nil {
			return nil, fmt.Errorf("hole %d of %d: %w", i+1, len(sites.Centers), err)
		}
		body = res
	}
	return body, nil
}

// holeCenter is the drill start point: an explicit center (projected onto the placement face's
// plane so the bore starts exactly at the surface), or the face centroid when none is set.
func holeCenter(def *HoleDefinition, face *topo.Face) math.Point3 {
	if def.Center == nil {
		return centroidOf(faceVertexPoints(face))
	}
	return projectOntoFacePlane(*def.Center, face)
}

// projectOntoFacePlane drops a model-space point onto a planar face's plane along its normal.
func projectOntoFacePlane(p math.Point3, face *topo.Face) math.Point3 {
	n, err := math.UnitVector3FromVector(face.Geometry().NormalAt(0, 0))
	if err != nil {
		return p
	}
	nv := n.AsVector()
	root := centroidOf(faceVertexPoints(face))
	d := root.VectorTo(p).Dot(nv)
	return p.TranslateBy(nv.Scale(-d))
}

// buildTool returns the clean drill cylinder a pattern replicates: the bore radius along the
// inward normal, spanning the body for a through hole or the requested depth for a blind one.
// It mirrors the cut the brep cutters apply (a counterbore's recess is omitted — the bore is
// the dominant geometry), so a pattern of the hole reproduces a sensible bore at each spot.
func (h *HoleFeature) buildTool(body *topo.Body, bore holeBore, into math.UnitVector3, r float64) *topo.Body {
	depth := bore.depth
	if h.def.ThroughAll {
		depth = throughDepth(body, bore.start, into)
	}
	return drillToolFrom(bore.start, into, r, depth, bore.entry, featOr(h.featName, "hole"))
}

// drill cuts the hole by its type: a counterbore is a shallow recess plus the bore, anything
// else a single cylinder. rec collects the faceted-fallback cuts' boolean diagnostics (#1601;
// nil discards; the exact-only countersink path never booleans, so it takes no recorder).
func (h *HoleFeature) drill(body *topo.Body, bore holeBore, into math.UnitVector3, r float64, rec *diag.Recorder) (*topo.Body, error) {
	center, depth, entry := bore.start, bore.depth, bore.entry
	switch h.def.Type {
	// A spotface IS a counterbore's recess — a flat-bottomed cylindrical seat — so it is cut by the
	// same builder; the seats stay distinct types because the distinction is real downstream (#1862).
	case CounterboreHole, SpotFaceHole:
		return h.cutCounterbore(body, center, into, r, depth, entry, rec)
	case CountersinkHole:
		return h.cutCountersink(body, center, into, r, depth)
	default:
		return h.cutDrilled(body, center, into, r, depth, entry, rec)
	}
}

// cutDrilled cuts a plain drilled hole: through, or blind with either a conical drill point
// (PointAngle > 0) or a flat bottom. A conical point that the part can't fit falls back to the
// flat/faceted blind cut.
func (h *HoleFeature) cutDrilled(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth, entry float64, rec *diag.Recorder) (*topo.Body, error) {
	if h.def.ThroughAll {
		return h.cutCylinder(body, center, into, r, depth, entry, true, rec)
	}
	if angle := callOrZero(h.def.PointAngle); angle > 0 {
		if res, err := brep.CutBlindConicalHole(body, center, into.AsVector(), r, depth, angle/2); err == nil {
			return res, nil
		}
	}
	return h.cutCylinder(body, center, into, r, depth, entry, false, rec)
}

// cutCountersink drills a conical countersink (recess widening to the sink diameter at the
// surface) above the bore. The exact builder produces a true cone wall; an unsupported part
// shape returns the error (a faceted cone approximation would be worse than a clear failure).
func (h *HoleFeature) cutCountersink(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth float64) (*topo.Body, error) {
	cr, angle := callOrZero(h.def.CounterDiameter)/2, callOrZero(h.def.CounterAngle)
	if cr <= r {
		return nil, fmt.Errorf("countersink: sink Ø %g must exceed bore Ø %g", 2*cr, 2*r)
	}
	if angle <= 0 || angle >= stdmath.Pi {
		return nil, fmt.Errorf("countersink: included angle %g must be in (0, π)", angle)
	}
	half := angle / 2
	depthCS := (cr - r) / stdmath.Tan(half)
	if !h.def.ThroughAll && depth <= depthCS {
		return nil, fmt.Errorf("countersink: total depth %g must exceed the sink depth %g", depth, depthCS)
	}
	return brep.CutCountersinkHole(body, center, into.AsVector(), r, depth-depthCS, cr, half, h.def.ThroughAll)
}

// cutCounterbore drills a counterbore: a shallow recess stepping down to the bore. The exact
// path (brep.CutCounterboreHole) builds the stepped result in one shot — two cylinder walls and
// an annular shoulder — from the planar slab; it does NOT chain two curved cuts (the second
// would feed a curved body to the planar-only boolean). The faceted fallback cuts the recess
// then the bore as sequential planar prisms (each stays planar, so it chains fine).
func (h *HoleFeature) cutCounterbore(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth, entry float64, rec *diag.Recorder) (*topo.Body, error) {
	cr, cd := callOrZero(h.def.CounterDiameter)/2, callOrZero(h.def.CounterDepth)
	if cr <= r {
		return nil, fmt.Errorf("counterbore: recess Ø %g must exceed bore Ø %g", 2*cr, 2*r)
	}
	if cd <= 0 || (!h.def.ThroughAll && depth <= cd) {
		return nil, fmt.Errorf("counterbore: recess depth %g must be > 0 and less than total depth %g", cd, depth)
	}
	if res, err := brep.CutCounterboreHole(body, center, into.AsVector(), r, depth-cd, cr, cd, h.def.ThroughAll); err == nil {
		return res, nil
	}
	return h.facetedCounterbore(body, center, into, r, depth, cr, cd, entry, rec)
}

// facetedCounterbore is the fallback for shapes the exact builder rejects: cut the recess prism,
// then the bore prism from the shoulder (both planar cuts, so they chain through the boolean).
func (h *HoleFeature) facetedCounterbore(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth, cr, cd, entry float64, rec *diag.Recorder) (*topo.Body, error) {
	stepped, err := ops.BooleanWithDiagnostics(ops.Cut, body, drillToolFrom(center, into, cr, cd, entry, featOr(h.featName, "hole")), rec)
	if err != nil {
		return nil, err
	}
	shoulder := center.TranslateBy(into.AsVector().Scale(math.Scalar(cd)))
	boreLen := depth - cd
	if h.def.ThroughAll {
		boreLen = throughDepth(stepped, shoulder, into)
	}
	return ops.BooleanWithDiagnostics(ops.Cut, stepped, drillTool(shoulder, into, r, boreLen, featOr(h.featName, "hole")), rec)
}

// cutCylinder cuts a single cylindrical hole, preferring an EXACT cylinder wall (K1b): a
// through hole via brep.CutCylindricalHole, a blind hole via brep.CutBlindCylindricalHole
// (wall + flat bottom). When the part shape isn't supported (the bore clips a face, or a blind
// depth would exit), it falls back to the faceted boolean — a through-cut when `through`, the
// requested depth otherwise.
func (h *HoleFeature) cutCylinder(body *topo.Body, center math.Point3, into math.UnitVector3, r, depth, entry float64, through bool, rec *diag.Recorder) (*topo.Body, error) {
	// A blind hole whose bottom reaches (or passes) the part's far extent along the axis IS a
	// through hole: the flush-bottom "blind" cut would leave a zero-thickness membrane for a
	// floor, which the exact blind drill rightly rejects — and the rejection used to fall to the
	// faceted prism cut, silently costing the bore its analytic cylinder wall (the tapped-hole
	// thread had nothing to attach to, Oblikovati#1693). Promote it to the through cut, matching
	// drill break-through behavior.
	if !through && depth >= throughDepth(body, center, into)-cutterOverhang-geom.ResolutionForBox(body.RangeBox()).Plane() {
		through = true
	}
	if through {
		if res, err := brep.CutCylindricalHole(body, center, into.AsVector(), r); err == nil {
			return res, nil
		}
		depth = throughDepth(body, center, into) // unsupported shape → faceted through-cut
	} else if res, err := brep.CutBlindCylindricalHole(body, center, into.AsVector(), r, depth); err == nil {
		return res, nil
	}
	tool := drillToolFrom(center, into, r, depth, entry, featOr(h.featName, "hole"))
	return ops.BooleanWithDiagnostics(ops.Cut, body, tool, rec)
}

// HoleFeatures and BossFeatures add hole/boss features into the engine.
type (
	HoleFeatures struct{ engine *PartFeatures }
	BossFeatures struct{ engine *PartFeatures }
)

// NewHoleFeatures / NewBossFeatures bind the collections to an engine.
func NewHoleFeatures(engine *PartFeatures) *HoleFeatures { return &HoleFeatures{engine} }
func NewBossFeatures(engine *PartFeatures) *BossFeatures { return &BossFeatures{engine} }

// AddDrilled adds a simple drilled hole on the placement face.
func (c *HoleFeatures) AddDrilled(faceKey []byte, diameter, depth func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole})
}

// AddDrilledThrough adds a drilled hole that goes all the way through the part (an exact
// cylinder wall on a planar-faced slab); depth is unused.
func (c *HoleFeatures) AddDrilledThrough(faceKey []byte, diameter func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: constFloat(0), ThroughAll: true, Type: DrilledHole})
}

// AddCounterbore adds a counterbore hole: a shallow recess (counterDiameter × counterDepth)
// at the placement face above the main bore (diameter × depth). Set ThroughAll on the returned
// definition for a through bore.
func (c *HoleFeatures) AddCounterbore(faceKey []byte, diameter, depth, counterDiameter, counterDepth func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{
		PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth,
		CounterDiameter: counterDiameter, CounterDepth: counterDepth, Type: CounterboreHole,
	})
}

// AddCountersink adds a countersink hole: a conical recess (sinkDiameter at the surface,
// included angle) above the main bore (diameter × depth). Set ThroughAll for a through bore.
func (c *HoleFeatures) AddCountersink(faceKey []byte, diameter, depth, sinkDiameter, includedAngle func() float64) *PartFeature {
	return c.addHole(&HoleDefinition{
		PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth,
		CounterDiameter: sinkDiameter, CounterAngle: includedAngle, Type: CountersinkHole,
	})
}

// AddTapped adds a tapped hole with thread data.
func (c *HoleFeatures) AddTapped(faceKey []byte, diameter, depth func() float64, designation string) *PartFeature {
	return c.addHole(&HoleDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Depth: depth, Type: DrilledHole, Tap: HoleTapInfo{Tapped: true, Designation: designation}})
}

// addHole adds a hole feature, naming it (Hole1, Hole2, …) so its generated topology has
// a stable, distinct lineage.
// resolveHoleFace finds the hole's placement face: by lineage key when present, else by a
// geometric descriptor (the externally-authored path, ADR-0040), else lost.
// centre, when given, is the drill point, used only as a fallback probe for a geometric selector
// that no longer matches exactly.
func resolveHoleFace(body *topo.Body, ref HoleFaceRef, centre *math.Point3) (*topo.Face, bool) {
	if len(ref.Key) > 0 {
		return body.FindFaceByKey(ref.Key)
	}
	if ref.Geom != nil {
		// Prefer the precise centroid+normal match. When it drifts past geomEdgeBindTol (an exporter
		// computing the centroid differently, a centroid shifted by later history, or an annular face
		// whose vertex-mean sits off the material), fall back to binding by a point on the placement
		// face's PLANE with the recorded normal. Try TWO such points, since either may miss:
		//   - the drill CENTRE, which lies on the face for a well-placed hole — but can sit on an
		//     adjacent curved face at a rim, off the target plane;
		//   - the recorded face CENTROID, which is coplanar with the placement face even when it
		//     falls just outside the face boundary (FindPlanarFaceThrough uses perpendicular distance
		//     to the plane, so an in-plane offset does not matter).
		if f, ok := body.FindFaceByGeometry(*ref.Geom, geomEdgeBindTol); ok {
			return f, true
		}
		if centre != nil {
			if f, ok := body.FindPlanarFaceThrough(*centre, ref.Geom.Normal, holeFaceThroughTol); ok {
				return f, true
			}
		}
		return body.FindPlanarFaceThrough(ref.Geom.Centroid, ref.Geom.Normal, holeFaceThroughTol)
	}
	return nil, false
}

// holeFaceThroughTol is how far the drill centre may sit off a candidate placement face's plane
// (perpendicular) and still bind it. The centre lies on the face by construction, so this only
// absorbs round-trip/recompute noise; the match is otherwise pinned by the outward normal.
const holeFaceThroughTol = math.Scalar(1e-2)

func (c *HoleFeatures) addHole(def *HoleDefinition) *PartFeature {
	hf := &HoleFeature{def: def}
	pf := c.engine.Add(hf)
	pf.SetName(c.engine.UniqueName("Hole"))
	hf.featName = pf.name
	return pf
}

// Add adds a cylindrical boss on the placement face, naming it (Boss1, Boss2, …) so its
// generated topology has a stable, distinct lineage.
func (c *BossFeatures) Add(faceKey []byte, diameter, height func() float64) *PartFeature {
	def := &BossDefinition{PlacementFaceKey: faceKey, Diameter: diameter, Height: height}
	// Capture the placement face's mint-time anchor against the running body so the geometric
	// recovery tier survives an upstream edit that renames the face (ADR-0043 P6 / #1579). Every
	// authoring path funnels here; the recipe restore uses addBoss, which preserves persisted
	// anchors and never recaptures (no doc-dirty churn on reopen).
	def.FaceAnchors = captureFaceAnchors(featuresTipBody(c.engine), [][]byte{faceKey})
	return c.addBoss(def)
}

// addBoss registers a boss from a fully-built definition without capturing anchors (the recipe
// restore path, which carries the persisted anchors of its own).
func (c *BossFeatures) addBoss(def *BossDefinition) *PartFeature {
	bf := &BossFeature{def: def}
	pf := c.engine.Add(bf)
	pf.SetName(c.engine.UniqueName("Boss"))
	bf.featName = pf.name
	return pf
}
