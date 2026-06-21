//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command siteshot is a throwaway live-capture driver that builds a representative machined
// part (base plate + boss + filleted corners + a through hole), dresses it with a color style
// and ground shadows, and saves marketing renders for the oblikovati.org website — either the
// clean 3D viewport or the whole application window (ribbon + browser + viewport). It reuses the
// production tools and DrawChrome render path, so the PNG is exactly what the app draws.
//
//	go run ./head/cmd/siteshot -shot part   -out /tmp/site-part.png
//	go run ./head/cmd/siteshot -shot window -out /tmp/site-window.png
package main

import (
	"flag"
	"fmt"
	stdmath "math"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

func main() {
	shot := flag.String("shot", "part", "part|window")
	out := flag.String("out", "/tmp/siteshot.png", "PNG output path")
	style := flag.String("style", "Brass", "color style to assign to the part")
	frames := flag.Int("frames", 10, "frames to render before capture")
	flag.Parse()
	if err := run(*shot, *out, *style, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "siteshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out)
}

func run(shot, out, style string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	def := buildBracket(s)
	dress(s, def, style)
	if err := s.SetViewOrientation(types.IsoTopRightViewOrientation, true); err != nil {
		return err
	}
	s.TickCameraAnimation(100)

	win, err := native.CreateWindow(1440, 900, "siteshot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	for i := 0; i < frames; i++ {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	if shot == "window" {
		return win.SaveWindowPNG(out)
	}
	return win.SaveViewportPNG(out)
}

// buildBracket models a base plate with a central boss, fillets every vertical corner, and
// bores a through hole down the boss — a small but recognizable machined part.
func buildBracket(s *app.Session) *compdef.PartComponentDefinition {
	def := newPart(s, "bracket.opd")
	plate := rect(def, -5, -3, 5, 3)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(plate, 0, ops.NewBody, func() float64 { return 1.2 })
	def.Recompute()

	boss := rect(def, -2.5, -1.5, 2.5, 1.5)
	def.Recompute()
	joinExtrude(s, boss, 3)

	body := def.SurfaceBodies().Item(0)
	filletVerticals(s, body, 0.8)

	drillThroughTop(s, def.SurfaceBodies().Item(0))
	s.CancelTool() // dismiss the committed tool's dialog so the window shot reads as a finished part
	return def
}

// joinExtrude extrudes the given profile up by h and unions it onto the running solid.
func joinExtrude(s *app.Session, sk *sketch.Sketch, h float64) {
	t := app.NewExtrudeTool()
	s.StartTool(t)
	t.Pick(s, app.ProfileHandle{Sketch: sk, ProfileIndex: 0})
	t.SetOperation(ops.Join)
	t.SetDistance(h)
	if err := t.Commit(s); err != nil {
		panic(err)
	}
}

// filletVerticals rounds every vertical corner edge of the body with the given radius.
func filletVerticals(s *app.Session, body *topo.Body, r float64) {
	t := app.NewFilletTool()
	s.StartTool(t)
	for _, e := range verticalEdges(body.Edges()) {
		t.Pick(s, app.EdgeHandle{Edge: e})
	}
	t.SetRadius(r)
	if err := t.Commit(s); err != nil {
		panic(err)
	}
}

// drillThroughTop bores a through hole centered on the boss's top face.
func drillThroughTop(s *app.Session, body *topo.Body) {
	t := app.NewHoleTool()
	s.StartTool(t)
	t.Pick(s, app.FaceHandle{Face: topmostZFace(body), Body: body})
	t.SetDiameter(1.6)
	t.SetDepth(10) // longer than the stack → through
	if err := t.Commit(s); err != nil {
		panic(err)
	}
}

// dress assigns a color style to the part and turns on a colored ground plane with shadows so
// the render reads as a presentation, not a wireframe test.
func dress(s *app.Session, def *compdef.PartComponentDefinition, style string) {
	if bodies := s.VisibleBodies(); len(bodies) > 0 && style != "" {
		_ = s.AssignColorStyleToBody(string(bodies[0].ReferenceKey()), style)
	}
	set := s.DisplaySettings(0)
	set.GroundPlane.Color = types.NewColor(28, 32, 44)
	set.GroundPlane.Visible = true
	s.SetDisplaySettings(0, set)
	sh := s.ShadowSettings()
	sh.GroundShadows = true
	s.SetShadowSettings(sh)
	_ = def
}

// --- geometry helpers ------------------------------------------------------------------

func newPart(s *app.Session, name string) *compdef.PartComponentDefinition {
	pd, err := compdef.AddPart(s.Workspace(), name, true)
	if err != nil {
		panic(err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	return pd.Content().(*compdef.PartComponentDefinition)
}

// rect adds an axis-aligned rectangle [x0,y0]-[x1,y1] on the XY plane.
func rect(def *compdef.PartComponentDefinition, x0, y0, x1, y1 float64) *sketch.Sketch {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(x0, y0))
	c1 := sk.Points().Add(math.P2(x1, y0))
	c2 := sk.Points().Add(math.P2(x1, y1))
	c3 := sk.Points().Add(math.P2(x0, y1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return sk
}

// verticalEdges returns every edge running mostly along Z (the part's corner edges).
func verticalEdges(edges []*topo.Edge) []*topo.Edge {
	var out []*topo.Edge
	for _, e := range edges {
		pts := ops.TessellateEdge(e, ops.DefaultQuality())
		if len(pts) < 2 {
			continue
		}
		a, b := pts[0], pts[len(pts)-1]
		dz := stdmath.Abs(a.Z - b.Z)
		if dz > stdmath.Abs(a.X-b.X) && dz > stdmath.Abs(a.Y-b.Y) {
			out = append(out, e)
		}
	}
	return out
}

// topmostZFace returns the +Z-facing face with the greatest centroid height (the boss top).
func topmostZFace(b *topo.Body) *topo.Face {
	var best *topo.Face
	bestZ := stdmath.Inf(-1)
	for _, f := range b.Faces() {
		if float64(f.Geometry().NormalAt(0, 0).Z) <= 0.9 {
			continue
		}
		z := faceCentroidZ(f)
		if z > bestZ {
			best, bestZ = f, z
		}
	}
	if best == nil {
		return b.Faces()[0]
	}
	return best
}

// faceCentroidZ averages the face's vertex heights — a cheap height key for face picking.
func faceCentroidZ(f *topo.Face) float64 {
	var sum float64
	var n int
	for _, v := range f.Vertices() {
		sum += float64(v.Point().Z)
		n++
	}
	if n == 0 {
		return stdmath.Inf(-1)
	}
	return sum / float64(n)
}
