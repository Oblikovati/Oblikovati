// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/topo"
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

// Recompute resolves the picked edges against the running body, then defers the
// rounding geometry.
func (f *FilletFeature) Recompute(in Input) (Output, error) {
	return resolveEdgesThenDefer(in, f.def.EdgeKeys, "fillet")
}

// ChamferDefinition bevels selected edges by a distance.
type ChamferDefinition struct {
	EdgeKeys [][]byte
	Distance func() float64
}

// ChamferFeature is an equal-distance edge chamfer.
type ChamferFeature struct{ def *ChamferDefinition }

func (c *ChamferFeature) Definition() *ChamferDefinition { return c.def }
func (c *ChamferFeature) Kind() string                   { return "chamfer" }
func (c *ChamferFeature) Recompute(in Input) (Output, error) {
	return resolveEdgesThenDefer(in, c.def.EdgeKeys, "chamfer")
}

// ShellDefinition hollows a body, removing the selected faces, to a wall thickness.
type ShellDefinition struct {
	RemovedFaceKeys [][]byte
	Thickness       func() float64
}

// ShellFeature hollows a solid.
type ShellFeature struct{ def *ShellDefinition }

func (s *ShellFeature) Definition() *ShellDefinition { return s.def }
func (s *ShellFeature) Kind() string                 { return "shell" }
func (s *ShellFeature) Recompute(in Input) (Output, error) {
	return resolveFacesThenDefer(in, s.def.RemovedFaceKeys, "shell")
}

// FaceDraftDefinition tapers selected faces by an angle about a pull direction.
type FaceDraftDefinition struct {
	FaceKeys [][]byte
	Angle    func() float64
}

// FaceDraftFeature applies draft to faces.
type FaceDraftFeature struct{ def *FaceDraftDefinition }

func (d *FaceDraftFeature) Definition() *FaceDraftDefinition { return d.def }
func (d *FaceDraftFeature) Kind() string                     { return "draft" }
func (d *FaceDraftFeature) Recompute(in Input) (Output, error) {
	return resolveFacesThenDefer(in, d.def.FaceKeys, "draft")
}

// ThreadDefinition applies thread data to a cylindrical face (cosmetic in phase A).
type ThreadDefinition struct {
	FaceKey     []byte
	Designation string
}

// ThreadFeature tags a cylindrical face with thread data.
type ThreadFeature struct{ def *ThreadDefinition }

func (t *ThreadFeature) Definition() *ThreadDefinition { return t.def }
func (t *ThreadFeature) Kind() string                  { return "thread" }
func (t *ThreadFeature) Recompute(in Input) (Output, error) {
	return resolveFacesThenDefer(in, [][]byte{t.def.FaceKey}, "thread")
}

// resolveEdgesThenDefer resolves edge keys against the running body and, if all
// bind, defers the geometry (passthrough + ErrDeferred); a lost key is a Sick error.
func resolveEdgesThenDefer(in Input, keys [][]byte, kind string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	for _, k := range keys {
		if _, ok := body.FindEdgeByKey(k); !ok {
			return Output{}, fmt.Errorf("%s: edge reference lost", kind)
		}
	}
	return Output{Bodies: in.Bodies}, ErrDeferred
}

// resolveFacesThenDefer is the face-input analogue.
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

// AddChamfer bevels the given edges by distance.
func (c *DressUpFeatures) AddChamfer(edgeKeys [][]byte, distance func() float64) *PartFeature {
	return c.engine.Add(&ChamferFeature{def: &ChamferDefinition{EdgeKeys: edgeKeys, Distance: distance}})
}

// AddShell hollows the body, removing the given faces, to thickness.
func (c *DressUpFeatures) AddShell(removedFaceKeys [][]byte, thickness func() float64) *PartFeature {
	return c.engine.Add(&ShellFeature{def: &ShellDefinition{RemovedFaceKeys: removedFaceKeys, Thickness: thickness}})
}

// AddDraft tapers the given faces by angle.
func (c *DressUpFeatures) AddDraft(faceKeys [][]byte, angle func() float64) *PartFeature {
	return c.engine.Add(&FaceDraftFeature{def: &FaceDraftDefinition{FaceKeys: faceKeys, Angle: angle}})
}

// AddThread tags a cylindrical face with thread data.
func (c *DressUpFeatures) AddThread(faceKey []byte, designation string) *PartFeature {
	return c.engine.Add(&ThreadFeature{def: &ThreadDefinition{FaceKey: faceKey, Designation: designation}})
}
