// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// Dress-up features operate on existing topology picked by the user — edges to
// round/chamfer, faces to shell/draft — held as reference keys (topo lineage keys,
// M07) and re-resolved against the running body each recompute via the topological-
// naming rebind. A picked edge "the same" after an upstream edit re-resolves; one
// that genuinely vanished makes the feature go Sick and surface for re-selection
// (parametric-cad §7, ADR-0010). The rounding/cut geometry itself is kernel phase B
// (rolling-ball fillets), so once inputs resolve these features report
// [ErrDeferred] (→ health.Warning) and pass the body through unchanged.

// FilletDefinition rounds selected edges to a radius.
type FilletDefinition struct {
	EdgeKeys [][]byte
	Radius   func() float64
}

// FilletFeature is a constant-radius edge fillet.
type FilletFeature struct{ def *FilletDefinition }

// Definition returns the fillet recipe.
func (f *FilletFeature) Definition() *FilletDefinition { return f.def }

// Kind implements [Feature].
func (f *FilletFeature) Kind() string { return "fillet" }

// Recompute rounds the picked convex edges on the running body with a real rolling-ball
// blend (cylinder faces). See fillet.go.
func (f *FilletFeature) Recompute(in Input) (Output, error) {
	return filletBody(in, f.def.EdgeKeys, callOrZero(f.def.Radius), "fillet")
}

// ChamferDefinition bevels selected edges by a distance. FlatCorners blends a vertex
// where three selected edges meet into a flat triangular face (the Inventor default);
// when false the three chamfer planes are left to meet at a point.
type ChamferDefinition struct {
	EdgeKeys    [][]byte
	Distance    func() float64
	FlatCorners bool
}

// ChamferFeature is an equal-distance edge chamfer.
type ChamferFeature struct {
	def      *ChamferDefinition
	featName string
}

func (c *ChamferFeature) Definition() *ChamferDefinition { return c.def }
func (c *ChamferFeature) Kind() string                   { return "chamfer" }

// Recompute bevels each selected (convex) edge by cutting a wedge tool along it via the
// boolean. See chamfer.go.
func (c *ChamferFeature) Recompute(in Input) (Output, error) {
	return chamferEdges(in, c.def.EdgeKeys, callOrZero(c.def.Distance), featOr(c.featName, "chamfer"), c.def.FlatCorners)
}

// ShellDefinition hollows a body, removing the selected faces, to a wall thickness.
type ShellDefinition struct {
	RemovedFaceKeys [][]byte
	Thickness       func() float64
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
	return shellBody(in, s.def.RemovedFaceKeys, callOrZero(s.def.Thickness), featOr(s.featName, "shell"))
}

// FaceDraftDefinition tapers selected faces by an angle about a pull direction.
type FaceDraftDefinition struct {
	FaceKeys [][]byte
	PullDir  math.Vector3
	Angle    func() float64
}

// FaceDraftFeature applies draft to faces.
type FaceDraftFeature struct{ def *FaceDraftDefinition }

func (d *FaceDraftFeature) Definition() *FaceDraftDefinition { return d.def }
func (d *FaceDraftFeature) Kind() string                     { return "draft" }

// Recompute tapers the picked faces about the pull direction by the angle (see draft.go).
func (d *FaceDraftFeature) Recompute(in Input) (Output, error) {
	return draftBody(in, d.def.FaceKeys, d.def.PullDir, callOrZero(d.def.Angle), "draft")
}

// ThreadDefinition applies thread data to a cylindrical face. Cut=false is a cosmetic thread
// (data + display, solid unchanged); Cut=true models a real thread (a helical groove cut).
type ThreadDefinition struct {
	FaceKey     []byte
	Designation string
	Cut         bool
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
// face makes the feature Sick.
func (t *ThreadFeature) Recompute(in Input) (Output, error) {
	spec, err := ParseThreadDesignation(t.def.Designation)
	if err != nil {
		return Output{}, err
	}
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	face, ok := body.FindFaceByKey(t.def.FaceKey)
	if !ok {
		return Output{}, fmt.Errorf("thread: face reference lost")
	}
	cyl, ok := face.Geometry().(geom.Cylinder)
	if !ok {
		return Output{}, fmt.Errorf("thread %q: face is not cylindrical (%T)", t.def.Designation, face.Geometry())
	}
	vMin, vMax := axialExtent(face.RangeBox(), cyl)
	spec.Internal = bodyHasMaterialOutside(body, cyl, (vMin+vMax)/2, (spec.MajorDiameter-spec.MinorDiameter)/2/10)
	t.spec = &spec
	if !t.def.Cut {
		return Output{Bodies: in.Bodies}, nil // cosmetic: solid unchanged
	}
	// Modeled (cut) thread: retype the cylindrical face to a threaded surface — O(1), no
	// boolean — so it tessellates and measures as real threaded geometry.
	threaded := geom.ThreadedCylinder{
		Cylinder: cyl, Pitch: spec.Pitch / 10, Depth: (spec.MajorDiameter - spec.MinorDiameter) / 2 / 10,
		Internal: spec.Internal, RightHanded: spec.RightHanded, VMin: vMin, VMax: vMax,
	}
	out := make([]*topo.Body, len(in.Bodies))
	copy(out, in.Bodies)
	threadedBody, err := ops.ReplaceFaceSurface(body, t.def.FaceKey, threaded)
	if err != nil {
		return Output{}, err
	}
	out[len(out)-1] = threadedBody // runningBody is the last body
	return Output{Bodies: out}, nil
}

// resolveFacesThenDefer resolves face keys against the running body and, if all bind, defers
// the geometry (passthrough + ErrDeferred); a lost key is a Sick error. (Still used by the
// thread cosmetic feature.)
func resolveFacesThenDefer(in Input, keys [][]byte, kind string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	for _, k := range keys {
		if _, ok := body.FindFaceByKey(k); !ok {
			return Output{}, fmt.Errorf("%s: face reference lost", kind)
		}
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

// AddFillet rounds the given edges (by reference key) to radius.
func (c *DressUpFeatures) AddFillet(edgeKeys [][]byte, radius func() float64) *PartFeature {
	return c.engine.Add(&FilletFeature{def: &FilletDefinition{EdgeKeys: edgeKeys, Radius: radius}})
}

// AddChamfer bevels the given edges by distance, blending three-edge corners flat (the
// default treatment). Use [AddChamferCorners] to choose the pointy corner instead.
func (c *DressUpFeatures) AddChamfer(edgeKeys [][]byte, distance func() float64) *PartFeature {
	return c.AddChamferCorners(edgeKeys, distance, true)
}

// AddChamferCorners bevels the given edges by distance; flatCorners selects whether a
// three-edge corner is blended into a flat triangular face (true) or left pointy (false).
func (c *DressUpFeatures) AddChamferCorners(edgeKeys [][]byte, distance func() float64, flatCorners bool) *PartFeature {
	cf := &ChamferFeature{def: &ChamferDefinition{EdgeKeys: edgeKeys, Distance: distance, FlatCorners: flatCorners}}
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

// AddDraftPull tapers the given faces by angle about an explicit pull direction.
func (c *DressUpFeatures) AddDraftPull(faceKeys [][]byte, pull math.Vector3, angle func() float64) *PartFeature {
	return c.engine.Add(&FaceDraftFeature{def: &FaceDraftDefinition{FaceKeys: faceKeys, PullDir: pull, Angle: angle}})
}

// AddThread tags a cylindrical face with thread data; cut=true models a real (cut) thread,
// cut=false a cosmetic one.
func (c *DressUpFeatures) AddThread(faceKey []byte, designation string, cut bool) *PartFeature {
	return c.engine.Add(&ThreadFeature{def: &ThreadDefinition{FaceKey: faceKey, Designation: designation, Cut: cut}})
}
