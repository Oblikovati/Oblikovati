// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// attachCapCloud adds a spherical-cap scan region cloud to the part and selects it.
func attachCapCloud(t *testing.T, s *Session, def *compdef.PartComponentDefinition) *pointcloud.PointCloud {
	t.Helper()
	const r, half, n = 10.0, 3.0, 12
	pts := make([]math.Point3, 0, n*n)
	for i := range n {
		for j := range n {
			x := -half + 2*half*float64(i)/float64(n-1)
			y := -half + 2*half*float64(j)/float64(n-1)
			pts = append(pts, math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(stdmath.Sqrt(r*r-x*x-y*y))))
		}
	}
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, pts)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	return pc
}

func TestFitSurfaceToolParams(t *testing.T) {
	t.Parallel()
	tool := NewFitSurfaceTool()
	if tool.Name() != "Fit Surface" {
		t.Fatal("name")
	}
	if tool.CanCommit() {
		t.Error("no cloud selected → not committable")
	}
	p := tool.Params()
	p.Ints[0].Set(2)
	p.Ints[1].Set(4)
	p.Ints[2].Set(5)
	if tool.degree != 2 || tool.nu != 4 || tool.nv != 5 {
		t.Errorf("params = {deg %d nu %d nv %d}, want {2 4 5}", tool.degree, tool.nu, tool.nv)
	}
}

func TestFitSurfaceToolCommitsAndReports(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	attachCapCloud(t, s, def)
	tool := NewFitSurfaceTool()
	tool.nu, tool.nv = 5, 5
	s.StartTool(tool)
	if !tool.CanCommit() {
		t.Fatal("a selected cloud with valid spans should be committable")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if tool.AddedFeature() == nil {
		t.Fatal("fit commit should add a feature")
	}
	r := tool.Report()
	if r == nil {
		t.Fatal("fit should report a deviation map")
	}
	if r.AbsMax > 0.5 { // a 5x5 bicubic fits a gentle cap tightly
		t.Errorf("fit |max| deviation %.4g too large", r.AbsMax)
	}
}

func TestFitSurfaceToolNeedsCloud(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	tool := NewFitSurfaceTool()
	s.StartTool(tool)
	if err := tool.Commit(s); err == nil {
		t.Error("commit without a selected cloud should error")
	}
}

func TestFitViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	attachCapCloud(t, s, def)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := s.Execute("Surface.FitToCloud"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Fit Surface" {
		t.Errorf("Surface.FitToCloud started %q, want Fit Surface", got)
	}
}

// TestFitSurfaceToolDraftFeature pins the #1626 commit-gate seam: no draft before a cloud is
// selected, a non-nil draft over the cloud's cropped points once commit-ready.
func TestFitSurfaceToolDraftFeature(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	tool := NewFitSurfaceTool()
	if _, ok := tool.DraftFeature(s); ok {
		t.Error("DraftFeature must not build before a cloud is selected")
	}
	attachCapCloud(t, s, def)
	tool.Start(s) // capture the selected cloud, as activation does
	if draft, ok := tool.DraftFeature(s); !ok || draft == nil {
		t.Fatalf("DraftFeature = (%v, %v), want a non-nil draft once a cloud is selected", draft, ok)
	}
}
