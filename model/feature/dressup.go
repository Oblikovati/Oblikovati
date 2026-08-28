// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
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
	return c.AddFilletDef(&FilletDefinition{EdgeKeys: edgeKeys, Radius: radius, CornerType: corner})
}

// AddFilletDef adds a fillet from a full definition — edge keys or edge sets, with the corner
// treatment, concave strategy and cross-section the caller wants — capturing mint-time edge
// anchors (ADR-0043 P6b) against the running body. It is the one authoring seam every other
// public builder funnels through; the recipe restore uses addFillet, which preserves the
// persisted anchors and never recaptures.
//
// It is exported because the cross-section and concave-strategy fields have no other builder:
// the Fillet tool used to create a plain fillet and then reach into the returned definition to
// set them, which meant the shipped path was not the one the unit tests exercised (#2052).
//
//	pf := dress.AddFilletDef(&feature.FilletDefinition{EdgeKeys: keys, Radius: r, CornerType: types.FilletCornerMiter})
func (c *DressUpFeatures) AddFilletDef(def *FilletDefinition) *PartFeature {
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
	return c.AddChamferDef(&ChamferDefinition{EdgeKeys: edgeKeys, Distance: distance, Type: types.ChamferDistance, FlatCorners: flatCorners})
}

// AddChamferDef adds a chamfer from a full definition — the setback mode with its second input,
// the corner treatment and the concave-edge strategy — capturing mint-time edge anchors
// (ADR-0043 P6b) against the running body. It is the one authoring seam every chamfer path uses;
// the recipe restore calls addChamfer directly so reopening a document never recaptures or
// rewrites anchors.
//
// It is exported because no per-mode builder carried every field: the Chamfer tool and the wire
// handler each created a chamfer and then reached into the returned definition to finish it, so
// the shipped path was not the one the builders described (#2045).
//
//	pf := dress.AddChamferDef(&feature.ChamferDefinition{EdgeKeys: keys, Distance: d, Type: types.ChamferDistance})
func (c *DressUpFeatures) AddChamferDef(def *ChamferDefinition) *PartFeature {
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
