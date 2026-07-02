// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/pointcloud"
)

// The Fit Surface tool (M36-F15) fits a clean Class-A NURBS surface to the selected point cloud's
// cropped region — the reverse-engineering move that turns scan data into editable styling geometry.
// Crop the cloud to the region first; the tool fits a degree×degree B-spline with the chosen U/V span
// (control) counts and reports the achieved deviation to the scan (F14) in the Command Window.

// FitSurfaceTool fits a NURBS surface to the selected cloud region.
type FitSurfaceTool struct {
	dialogTool
	cloud  *pointcloud.PointCloud
	degree int
	nu, nv int
	added  *feature.PartFeature
	report *analysis.DeviationReport
}

// NewFitSurfaceTool returns a fit tool defaulting to a bicubic 6×6 patch.
func NewFitSurfaceTool() *FitSurfaceTool {
	return &FitSurfaceTool{degree: feature.DefaultFitDegree, nu: feature.DefaultFitSpans, nv: feature.DefaultFitSpans}
}

// Name implements [Tool].
func (t *FitSurfaceTool) Name() string { return "Fit Surface" }

// Start captures the cloud selected in the browser (the region to fit).
func (t *FitSurfaceTool) Start(s *Session) { t.cloud, _ = s.SelectedPointCloud() }

// Prompt guides the input.
func (t *FitSurfaceTool) Prompt(*Session) string {
	return "Fit Surface: select a (cropped) point cloud, set degree and U/V spans, then OK."
}

// Params exposes the degree and U/V control-span counts.
func (t *FitSurfaceTool) Params() ToolParams {
	return ToolParams{Ints: []IntParam{
		{Label: "Degree", Get: func() int { return t.degree }, Set: func(v int) { t.degree = v }},
		{Label: "Spans U", Get: func() int { return t.nu }, Set: func(v int) { t.nu = v }},
		{Label: "Spans V", Get: func() int { return t.nv }, Set: func(v int) { t.nv = v }},
	}}
}

// CanCommit reports whether a cloud is selected and the control counts exceed the degree each way.
func (t *FitSurfaceTool) CanCommit() bool {
	return t.cloud != nil && t.degree >= 1 && t.nu > t.degree && t.nv > t.degree
}

// addFitSurface builds the fit feature over the cloud's cropped points into fs — the shared
// constructor used by both Commit (the part's engine) and DraftFeature (a scratch engine), so
// the two cannot drift.
func (t *FitSurfaceTool) addFitSurface(fs *feature.PartFeatures, pts []math.Point3) *feature.PartFeature {
	return feature.NewFitFeatures(fs).Add(pts, t.degree, t.nu, t.nv)
}

// DraftFeature implements [PartFeatureTool] (#1626): the fitted surface it would commit, built
// into a scratch engine so the commit gate and preview can evaluate it without touching the
// part. CanCommit guarantees the cloud is selected before the points are read.
func (t *FitSurfaceTool) DraftFeature(*Session) (feature.Feature, bool) {
	if !t.CanCommit() {
		return nil, false
	}
	pts := t.cloud.CroppedModelPoints()
	return draftFromScratch(func(fs *feature.PartFeatures) (*feature.PartFeature, error) {
		return t.addFitSurface(fs, pts), nil
	})
}

// Commit fits the surface to the cloud region, recomputes, and reports the deviation to the scan.
func (t *FitSurfaceTool) Commit(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if t.cloud == nil {
		return errors.New("fit surface: select a point cloud region first")
	}
	pts := t.cloud.CroppedModelPoints()
	t.added = t.addFitSurface(part.Features(), pts)
	part.Recompute()
	s.recordEdit(part, "Fit Surface")
	if !t.added.Health().OK() {
		return errors.New("fit surface: " + t.added.Health().Reason)
	}
	t.reportDeviation(s, part, pts)
	return nil
}

// reportDeviation maps the fitted surface against the scan region (F14) and feeds the summary.
func (t *FitSurfaceTool) reportDeviation(s *Session, part *compdef.PartComponentDefinition, pts []math.Point3) {
	bodies := part.SurfaceBodies()
	if bodies.Count() == 0 {
		return
	}
	surf, ok := nurbsFaceSurface(bodies.Item(bodies.Count() - 1))
	if !ok {
		return
	}
	r := analysis.SurfaceDeviationToPoints(surf, pts, deviationGrid, deviationGrid)
	t.report = &r
	s.feedNotice(fmt.Sprintf("Fitted surface deviation to scan: |max| %.4g, RMS %.4g over %d points", r.AbsMax, r.RMS, len(pts)))
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *FitSurfaceTool) AddedFeature() *feature.PartFeature { return t.added }

// Report returns the last fit deviation report (for inspection/tests).
func (t *FitSurfaceTool) Report() *analysis.DeviationReport { return t.report }
