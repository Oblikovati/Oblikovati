//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command previewshot is a throwaway live-capture driver for the in-canvas feature preview:
// it opens the real native window, sets up a part with a pending (uncommitted) feature of the
// requested kind, runs the production DrawChrome frame loop, and saves the 3D viewport — so the
// PNG shows the translucent result ghost exactly as the app draws it (green where the feature
// adds material, red where it removes it).
//
//	go run ./head/cmd/previewshot -op join   -out /tmp/p-join.png
//	go run ./head/cmd/previewshot -op cut    -out /tmp/p-cut.png
//	go run ./head/cmd/previewshot -op revolve -out /tmp/p-revolve.png   # join|cut|revolve|
//	  sweep|loft|coil|hole|fillet|chamfer|shell|draft|thread
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
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

func main() {
	op := flag.String("op", "cut", "pending feature to preview: join|cut|revolve|sweep|loft|coil|hole|fillet|chamfer|shell|draft|thread")
	out := flag.String("out", "/tmp/previewshot.png", "viewport PNG output path")
	frames := flag.Int("frames", 8, "frames to render before capture")
	flag.Parse()
	if err := run(*op, *out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "previewshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out)
}

func run(op, out string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	if err := setupOp(s, op); err != nil {
		return err
	}
	// Edge dress-ups (fillet/chamfer) set their own camera aimed at the modified corner so the
	// thin wedge faces us; everything else uses the standard iso view.
	if op != "fillet" && op != "chamfer" {
		if err := s.SetViewOrientation(types.IsoTopRightViewOrientation, true); err != nil {
			return err
		}
	}
	s.TickCameraAnimation(100)

	win, err := native.CreateWindow(1280, 800, "previewshot")
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
	return win.SaveViewportPNG(out)
}

// setupOp builds the part and starts the requested feature tool, leaving it pending so the
// viewport previews it.
func setupOp(s *app.Session, op string) error {
	switch op {
	case "join", "cut":
		startPendingExtrude(s, mustBaseBox(s), op)
	case "fillet":
		startPendingFillet(s, mustBaseBox(s))
	case "chamfer":
		startPendingChamfer(s, mustBaseBox(s))
	case "shell":
		startPendingShell(s, mustBlock(s))
	case "draft":
		startPendingDraft(s, mustBlock(s))
	case "hole":
		startPendingHole(s, mustBlock(s))
	case "revolve":
		startPendingRevolve(s)
	case "sweep":
		startPendingSweep(s)
	case "loft":
		startPendingLoft(s)
	case "coil":
		startPendingCoil(s)
	case "thread":
		return startPendingThread(s)
	case "faceoffset":
		startPendingFaceOffset(s, mustBlock(s))
	case "deleteface":
		startPendingDeleteFace(s, mustBlock(s))
	default:
		return fmt.Errorf("unknown -op %q", op)
	}
	return nil
}

// --- base solids -----------------------------------------------------------------------

// newPart adds and activates an empty part, returning its definition.
func newPart(s *app.Session, name string) *compdef.PartComponentDefinition {
	pd, err := compdef.AddPart(s.Workspace(), name, true)
	if err != nil {
		panic(err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	return pd.Content().(*compdef.PartComponentDefinition)
}

// mustBaseBox commits a 6×6×6 base cube (the target for join/cut/fillet/chamfer).
func mustBaseBox(s *app.Session) *compdef.PartComponentDefinition { return buildBaseBox(s) }

func buildBaseBox(s *app.Session) *compdef.PartComponentDefinition {
	def := newPart(s, "previewshot.opd")
	sk := addSquare(def, 0, 0, 6)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 6 })
	def.Recompute()
	return def
}

// mustBlock commits a 6×6×3 base solid (the target for shell/draft/hole).
func mustBlock(s *app.Session) *compdef.PartComponentDefinition {
	def := newPart(s, "previewshot.opd")
	sk := addSquare(def, 0, 0, 6)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 3 })
	def.Recompute()
	return def
}

// addSquare adds a side×side square sketch on the XY plane with its corner at (ox,oy).
func addSquare(def *compdef.PartComponentDefinition, ox, oy, side float64) *sketch.Sketch {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(ox, oy))
	c1 := sk.Points().Add(math.P2(ox+side, oy))
	c2 := sk.Points().Add(math.P2(ox+side, oy+side))
	c3 := sk.Points().Add(math.P2(ox, oy+side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return sk
}

// offsetSquare adds a side×side square on XY whose lower-left corner is at (x0,0).
func offsetSquare(def *compdef.PartComponentDefinition, x0, side float64) app.ProfileHandle {
	return app.ProfileHandle{Sketch: addSquare(def, x0, 0, side), ProfileIndex: 0}
}

// --- swept-solid features (tool body → green) ------------------------------------------

func startPendingExtrude(s *app.Session, def *compdef.PartComponentDefinition, op string) {
	sk := addSquare(def, 2, 2, 4)
	def.Recompute()
	ext := app.NewExtrudeTool()
	s.StartTool(ext)
	ext.Pick(s, app.ProfileHandle{Sketch: sk, ProfileIndex: 0})
	if op == "join" {
		ext.SetOperation(ops.Join)
		ext.SetDistance(8) // a column standing proud above the base → green
	} else {
		ext.SetOperation(ops.Cut)
		ext.SetDistance(4) // a hole bored through the base → red
	}
}

func startPendingRevolve(s *app.Session) {
	def := newPart(s, "previewshot.opd")
	prof := offsetSquare(def, 3, 2) // 2×2 square 3 units off the Y axis → a ring
	def.Recompute()
	rv := app.NewRevolveTool()
	s.StartTool(rv)
	rv.Pick(s, prof)
	rv.SetAxis(feature.OriginYAxis)
	rv.SetAngle(3 * stdmath.Pi / 2) // 270° so the ring's opening is visible
}

func startPendingSweep(s *app.Session) {
	def := newPart(s, "previewshot.opd")
	prof := app.ProfileHandle{Sketch: centeredSquare(def, sketch.XYPlane(), 1), ProfileIndex: 0}
	pathSk := def.Sketches().Add(sketch.XZPlane())
	a := pathSk.Points().Add(math.P2(0, 0))
	b := pathSk.Points().Add(math.P2(0, 6))
	pathSk.Lines().Add(a, b)
	def.Recompute()
	sw := app.NewSweepTool()
	s.StartTool(sw)
	sw.Pick(s, prof)
	sw.Pick(s, app.PathHandle{Sketch: pathSk, PathIndex: 0})
}

func startPendingLoft(s *app.Session) {
	def := newPart(s, "previewshot.opd")
	bottom := app.ProfileHandle{Sketch: centeredSquare(def, sketch.XYPlane(), 3), ProfileIndex: 0}
	top := app.ProfileHandle{Sketch: centeredSquare(def, planeAtZ(6), 1), ProfileIndex: 0}
	def.Recompute()
	l := app.NewLoftTool()
	s.StartTool(l)
	l.Pick(s, bottom)
	l.Pick(s, top)
}

func startPendingCoil(s *app.Session) {
	def := newPart(s, "previewshot.opd")
	prof := offsetSquare(def, 3, 1) // 1×1 square 3 off the Y axis → a helical spring
	def.Recompute()
	c := app.NewCoilTool()
	s.StartTool(c)
	c.Pick(s, prof)
	c.SetAxis(feature.OriginYAxis)
	c.SetPitch(2)
	c.SetRevolutions(3)
}

// --- hole (drill tool body → red) ------------------------------------------------------

func startPendingHole(s *app.Session, def *compdef.PartComponentDefinition) {
	body := def.SurfaceBodies().Item(0)
	hole := app.NewHoleTool()
	s.StartTool(hole)
	hole.Pick(s, app.FaceHandle{Face: faceByNormalZ(body, 1), Body: body})
	hole.SetDiameter(2)
	hole.SetDepth(4) // through the 3-thick block → red drill
}

// --- dress-ups (changed-faces diff) ----------------------------------------------------

func startPendingFillet(s *app.Session, def *compdef.PartComponentDefinition) {
	body := def.SurfaceBodies().Item(0)
	edge := verticalEdge(body.Edges())
	fillet := app.NewFilletTool()
	s.StartTool(fillet)
	fillet.Pick(s, app.EdgeHandle{Edge: edge})
	fillet.SetRadius(1.2)
	aimCameraAtEdge(s, body, edge)
}

func startPendingChamfer(s *app.Session, def *compdef.PartComponentDefinition) {
	body := def.SurfaceBodies().Item(0)
	edge := verticalEdge(body.Edges())
	ch := app.NewChamferTool()
	s.StartTool(ch)
	ch.Pick(s, app.EdgeHandle{Edge: edge})
	ch.SetDistance(1.2)
	aimCameraAtEdge(s, body, edge)
}

// aimCameraAtEdge frames the body from outside the given edge (looking at its midpoint along
// the outward direction, tilted down), so a thin edge wedge faces the camera rather than
// presenting edge-on.
func aimCameraAtEdge(s *app.Session, body *topo.Body, e *topo.Edge) {
	pts := ops.TessellateEdge(e, ops.DefaultQuality())
	mid := pts[len(pts)/2]
	rb := body.RangeBox()
	cx, cy := (rb.Min.X+rb.Max.X)/2, (rb.Min.Y+rb.Max.Y)/2
	ox, oy := mid.X-cx, mid.Y-cy
	n := stdmath.Hypot(ox, oy)
	if n == 0 {
		n = 1
	}
	dist := (rb.Max.X - rb.Min.X) * 2.4
	cam := scene.NewCamera(1280, 800)
	cam.Target = mid
	cam.Eye = math.P3(mid.X+ox/n*dist, mid.Y+oy/n*dist, mid.Z+dist*0.5)
	cam.Up = math.V3(0, 0, 1)
	s.SetCamera(cam)
}

func startPendingShell(s *app.Session, def *compdef.PartComponentDefinition) {
	body := def.SurfaceBodies().Item(0)
	sh := app.NewShellTool()
	s.StartTool(sh)
	sh.Pick(s, app.FaceHandle{Face: faceByNormalZ(body, 1), Body: body}) // open the top
	sh.SetThickness(0.6)
}

func startPendingDraft(s *app.Session, def *compdef.PartComponentDefinition) {
	body := def.SurfaceBodies().Item(0)
	d := app.NewDraftTool()
	s.StartTool(d)
	d.Pick(s, app.FaceHandle{Face: faceByNormalX(body, 1), Body: body})
	d.SetAngleDegrees(12)
}

func startPendingThread(s *app.Session) error {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1.0, 5.0)
	if err != nil {
		return err
	}
	def := newPart(s, "previewshot.opd")
	feature.NewBaseFeatures(def.Features()).AddBase(cyl)
	def.Recompute()
	body := def.SurfaceBodies().Item(0)
	tool := app.NewThreadTool()
	s.StartTool(tool)
	tool.Pick(s, app.FaceHandle{Face: cylindricalFace(body), Body: body})
	tool.SetStandardIndex(0)
	tool.SetSizeIndex(6) // M8
	tool.SetPitchIndex(0)
	tool.SetCut(true) // cut thread → removes material → red
	return nil
}

// startPendingFaceOffset offsets the block's top face outward (adds material → green delta).
func startPendingFaceOffset(s *app.Session, def *compdef.PartComponentDefinition) {
	body := def.SurfaceBodies().Item(0)
	t := app.NewFaceOffsetTool()
	s.StartTool(t)
	t.Pick(s, app.FaceHandle{Face: faceByNormalZ(body, 1), Body: body})
	t.SetDistance(2)
}

// startPendingDeleteFace deletes (and heals) the block's top face (removes material → red delta).
func startPendingDeleteFace(s *app.Session, def *compdef.PartComponentDefinition) {
	body := def.SurfaceBodies().Item(0)
	t := app.NewDeleteFaceTool()
	s.StartTool(t)
	t.Pick(s, app.FaceHandle{Face: faceByNormalZ(body, 1), Body: body})
}

// --- geometry helpers ------------------------------------------------------------------

func centeredSquare(def *compdef.PartComponentDefinition, plane sketch.Plane, half float64) *sketch.Sketch {
	sk := def.Sketches().Add(plane)
	c0 := sk.Points().Add(math.P2(-half, -half))
	c1 := sk.Points().Add(math.P2(half, -half))
	c2 := sk.Points().Add(math.P2(half, half))
	c3 := sk.Points().Add(math.P2(-half, half))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return sk
}

func planeAtZ(z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(0, 0, z), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	return p
}

// faceByNormalZ / faceByNormalX return the body's first face whose outward normal points
// along ±Z / ±X (sign = +1 or -1).
func faceByNormalZ(b *topo.Body, sign float64) *topo.Face {
	for _, f := range b.Faces() {
		if sign*float64(f.Geometry().NormalAt(0, 0).Z) > 0.9 {
			return f
		}
	}
	return b.Faces()[0]
}

func faceByNormalX(b *topo.Body, sign float64) *topo.Face {
	for _, f := range b.Faces() {
		if sign*float64(f.Geometry().NormalAt(0, 0).X) > 0.9 {
			return f
		}
	}
	return b.Faces()[0]
}

func cylindricalFace(b *topo.Body) *topo.Face {
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f
		}
	}
	return b.Faces()[0]
}

// verticalEdge returns the first edge running mostly along Z (a box's vertical corner).
func verticalEdge(edges []*topo.Edge) *topo.Edge {
	for _, e := range edges {
		pts := ops.TessellateEdge(e, ops.DefaultQuality())
		if len(pts) < 2 {
			continue
		}
		a, b := pts[0], pts[len(pts)-1]
		dz := abs(a.Z - b.Z)
		if dz > abs(a.X-b.X) && dz > abs(a.Y-b.Y) {
			return e
		}
	}
	return edges[0]
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
