// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// The Fit-Surface feature (M36-F15) fits a clean Class-A NURBS surface to a region of scanned data —
// the reverse-engineering step that turns a point cloud / mesh region into editable styling geometry.
// The region points are stored baked into the recipe (the tool bakes them from the cloud's cropped,
// model-space points); the feature least-squares fits a degree×degree B-spline with the requested
// nu×nv control net via ops.FitSurfaceToPoints and appends it as a surface body.

// Class-A fit defaults shared by the Fit Surface tool and the fitSurface MCP op: a bicubic patch
// (degree 3) with a 6×6 control net — few even spans for clean reflection lines.
const (
	DefaultFitDegree = 3
	DefaultFitSpans  = 6
)

// FitDefinition is the recipe for a fitted surface: the region points and the degree/span targets.
type FitDefinition struct {
	Points []math.Point3
	Degree int
	NU, NV int
}

// FitFeature fits a NURBS surface to a region of scan points.
type FitFeature struct {
	def      *FitDefinition
	featName string
}

// Definition returns the fit recipe.
func (f *FitFeature) Definition() *FitDefinition { return f.def }

// Kind implements [Feature].
func (f *FitFeature) Kind() string { return "fit-surface" }

// Recompute fits the surface and appends it as a surface body. It errors (→ sick) without enough
// region points or when the points do not determine a base plane.
func (f *FitFeature) Recompute(in Input) (Output, error) {
	if len(f.def.Points) < f.def.NU*f.def.NV {
		return Output{}, fmt.Errorf("fit surface: %d region points is fewer than the %dx%d control net", len(f.def.Points), f.def.NU, f.def.NV)
	}
	body, err := ops.FitSurfaceToPoints(f.def.Points, f.def.Degree, f.def.NU, f.def.NV)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: appendBody(in.Bodies, body)}, nil
}

// FitFeatures adds fit-surface features into the engine.
type FitFeatures struct{ engine *PartFeatures }

// NewFitFeatures binds the collection to a feature engine.
func NewFitFeatures(engine *PartFeatures) *FitFeatures { return &FitFeatures{engine: engine} }

// Add fits a NURBS surface to the region points with the given degree and nu×nv control net.
func (c *FitFeatures) Add(points []math.Point3, degree, nu, nv int) *PartFeature {
	def := &FitDefinition{Points: points, Degree: degree, NU: nu, NV: nv}
	ff := &FitFeature{def: def, featName: "Fit Surface"}
	pf := c.engine.Add(ff)
	ff.featName = pf.name
	return pf
}
