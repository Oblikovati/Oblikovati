//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Render validation: prove that BOTH a part document and an assembly document actually put their
// solid geometry on screen. This is the regression guard from the empty-assembly-viewport
// investigation — the model and render path were correct, but assemblies must KEEP rendering. The
// test renders each document offscreen through the real viewport panel and asserts a large fraction
// of bright body pixels, with an empty part as the blank baseline so the metric is shown to
// discriminate (and the threshold is meaningful, not arbitrary).
package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// buildBox extrudes a 4×4×5 solid into def (an origin-centred square pulled +Z), big enough that
// the framing camera sees it fill much of the viewport.
func buildBox(def *compdef.PartComponentDefinition) {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -2))
	c1 := sk.Points().Add(math.P2(2, -2))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(-2, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
}

// framingCamera looks down −Z at the box from a distance that frames it large in the viewport.
func framingCamera() scene.Camera {
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 12), math.P3(0, 0, 2.5), math.V3(0, 1, 0)
	return cam
}

// boxPart returns a session whose active document is a PART holding the box.
func rvBoxPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	buildBox(pd.Content().(*compdef.PartComponentDefinition))
	s.SetCamera(framingCamera())
	return s
}

// emptyPart returns a session whose active document is a PART with no geometry — the blank baseline
// that proves the lit-fraction metric reads ~0 on a viewport with nothing to draw.
func emptyPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "empty.opd", true); err != nil {
		t.Fatalf("add part: %v", err)
	}
	s.SetCamera(framingCamera())
	return s
}

// boxAssembly returns a session whose active document is an ASSEMBLY with one box component placed
// at identity — exercising the multi-document render path (VisibleInstances → assemblyInstances →
// instanced draw), the path that rendered blank in the investigation.
func boxAssembly(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	comp, err := compdef.AddPart(s.Workspace(), "comp.opd", true)
	if err != nil {
		t.Fatalf("add component part: %v", err)
	}
	buildBox(comp.Content().(*compdef.PartComponentDefinition))
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.opd", true)
	if err != nil {
		t.Fatalf("add assembly: %v", err)
	}
	asm := asmDoc.Content().(*compdef.AssemblyComponentDefinition)
	if _, err := asm.PlaceComponentFromFile(asmDoc, comp, "Box:1", math.Identity4()); err != nil {
		t.Fatalf("place component: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	s.SetCamera(framingCamera())
	return s
}

// litFraction renders several viewport frames into slot 0 (warming a cold instanced atlas) and
// returns the fraction of pixels that are bright body geometry — luminance well above the dark
// themed background. A blank viewport returns ~0; a solid filling the view returns a large fraction.
func litFraction(win *native.Window, s *app.Session) float64 {
	for i := 0; i < 8; i++ {
		viewportFrame(win, s)
	}
	px, w, h, ok := win.ReadbackViewport(0)
	if !ok || w == 0 || h == 0 {
		return 0
	}
	bright, total := 0, w*h
	for i := 0; i+3 < len(px); i += 4 {
		// BGRA bytes; ITU-R 601 luma. The light-gray body clears ~175; the blue background does not.
		b, g, r := float64(px[i]), float64(px[i+1]), float64(px[i+2])
		if 0.299*r+0.587*g+0.114*b > 175 {
			bright++
		}
	}
	return float64(bright) / float64(total)
}

// TestViewportRendersPartAndAssemblyGeometry asserts BOTH document kinds draw their solids. The
// empty baseline must read ~blank (so the metric discriminates), while the part and the assembly
// must each fill a large fraction — a blank assembly (the investigated bug) fails the last check.
func TestViewportRendersPartAndAssemblyGeometry(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()

	empty := litFraction(win, emptyPart(t))
	part := litFraction(win, rvBoxPart(t))
	asm := litFraction(win, boxAssembly(t))
	t.Logf("lit fraction: empty=%.4f part=%.4f assembly=%.4f", empty, part, asm)

	if empty > 0.02 {
		t.Fatalf("empty viewport should read ~blank, got lit fraction %.4f — metric not discriminating", empty)
	}
	if part < 0.10 {
		t.Errorf("PART viewport rendered blank: lit fraction %.4f, want a solid filling the view", part)
	}
	if asm < 0.10 {
		t.Errorf("ASSEMBLY viewport rendered blank: lit fraction %.4f — placed occurrence geometry is missing", asm)
	}
}
