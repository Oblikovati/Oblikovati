//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command motorshot is a throwaway live-capture driver: it runs the real Motor Designer
// add-in engine in-process (over an in-proc router caller), generating a Motor assembly of
// three coaxial parts (stator, rotor, magnets), gives each part a distinct appearance so the
// assembly reads as an assembly, and saves marketing renders for oblikovati.org.
//
//	go run ./head/cmd/motorshot -shot assembly -out /tmp/motor-assembly.png
//	go run ./head/cmd/motorshot -shot window   -out /tmp/motor-window.png
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/addin/router"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/material"
	designer "oblikovati.org/motor-designer/designer"
)

// partAppearance is the distinct appearance each motor occurrence (by name) gets, so the
// assembly reads as three different components. Stator and rotor are separate part documents
// but their placed bodies share a reference key, so a session-global color style cannot tell
// them apart — a PART-scoped appearance lives in each part's own assignment store, which does
// not collide. Steel/steel/magnet is also physically what the parts are.
var partAppearance = []struct {
	occurrence, name, hex string
	metallic, roughness   float32
}{
	{"Stator", "Motor Stator Steel", "#9aa2b0ff", 0.85, 0.40},
	{"Rotor", "Motor Rotor Steel", "#b8923eff", 0.88, 0.34},
	{"Magnets", "Motor Magnet", "#c2543eff", 0.25, 0.48},
}

// dressMotor gives each motor part a distinct PART-scoped appearance: it builds the
// appearances in the project library, then activates each source part document and assigns
// its appearance, so the live assembly render shows stator, rotor and magnets in different
// finishes (the assembly surface lookup resolves each occurrence through its source part's
// own assignment store, #1103).
func dressMotor(s *app.Session) error {
	asmDoc := s.Workspace().ActiveDocument()
	asm, ok := asmDoc.Content().(*compdef.AssemblyComponentDefinition)
	if !ok {
		return fmt.Errorf("active document %q is not an assembly", asmDoc.DisplayName())
	}
	wantByPart, err := appearancesByPart(s, asm)
	if err != nil {
		return err
	}
	for _, d := range s.Workspace().Documents() {
		part, ok := d.Content().(*compdef.PartComponentDefinition)
		if !ok {
			continue
		}
		apprID, ok := wantByPart[part]
		if !ok {
			continue
		}
		if err := s.Workspace().SetActiveDocument(d); err != nil {
			return err
		}
		if err := s.AssignAppearance(app.ScopePart, "", apprID); err != nil {
			return err
		}
	}
	return s.Workspace().SetActiveDocument(asmDoc)
}

// appearancesByPart builds the per-occurrence appearances and maps each source part definition
// to the appearance id it should wear, by matching placed occurrences to partAppearance names.
func appearancesByPart(s *app.Session, asm *compdef.AssemblyComponentDefinition) (map[*compdef.PartComponentDefinition]string, error) {
	byOcc := map[string]string{}
	for _, pa := range partAppearance {
		id, err := makeAppearance(s, pa.name, pa.hex, pa.metallic, pa.roughness)
		if err != nil {
			return nil, err
		}
		byOcc[pa.occurrence] = id
	}
	out := map[*compdef.PartComponentDefinition]string{}
	for _, pb := range asm.PlacedBodies() {
		id, ok := byOcc[pb.Source.Name()]
		if !ok {
			continue
		}
		if part, ok := pb.Source.Definition().(*compdef.PartComponentDefinition); ok {
			out[part] = id
		}
	}
	return out, nil
}

// makeAppearance creates a project-scoped PBR appearance with the given albedo (hex) and
// metallic/roughness, returning its id for assignment.
func makeAppearance(s *app.Session, name, hex string, metallic, roughness float32) (string, error) {
	a, err := s.DuplicateAppearance(material.DefaultAppearanceID, name)
	if err != nil {
		return "", err
	}
	albedo, err := types.ParseHex(hex)
	if err != nil {
		return "", err
	}
	black, _ := types.ParseHex("#000000ff")
	s.UpdateAppearance(a.ID(), material.AppearanceSpec{
		DisplayName: name, Albedo: albedo, Metallic: metallic, Roughness: roughness,
		Emissive: black, Opacity: 1,
	})
	return a.ID(), nil
}

// inProcCaller routes the add-in engine's host calls straight into the app router, exactly as
// the live head does (startAddIns), but without the C ABI — so the real Generate runs here.
type inProcCaller struct {
	rtr *router.Router
	s   *app.Session
}

func (c inProcCaller) Call(method string, req []byte) ([]byte, error) {
	return c.rtr.Handle(c.s, method, req)
}

func main() {
	shot := flag.String("shot", "assembly", "assembly|window")
	out := flag.String("out", "/tmp/motorshot.png", "PNG output path")
	frames := flag.Int("frames", 12, "frames to render before capture")
	flag.Parse()
	if err := run(*shot, *out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "motorshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out)
}

func run(shot, out string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	eng := designer.NewEngine(inProcCaller{rtr: router.New(opregistry.Default()), s: s})
	res, err := eng.Generate(designer.DefaultSpec())
	if err != nil {
		return fmt.Errorf("generate motor: %w", err)
	}
	fmt.Fprintf(os.Stderr, "motor: assembly=%d stator=%d rotor=%d magnets=%d (%d magnet bodies)\n",
		res.AssemblyID, res.StatorDocID, res.RotorDocID, res.MagnetDocID, res.MagnetBodies)

	if shot == "drawing" {
		if err := buildStatorDrawing(s); err != nil {
			return fmt.Errorf("build drawing: %w", err)
		}
	} else {
		if err := dressMotor(s); err != nil {
			return fmt.Errorf("dress motor: %w", err)
		}
		if err := s.SetViewOrientation(types.IsoTopRightViewOrientation, true); err != nil {
			return err
		}
		s.TickCameraAnimation(100)
	}

	win, err := native.CreateWindow(1440, 900, "motorshot")
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
	if shot == "assembly" {
		return win.SaveViewportPNG(out)
	}
	return win.SaveWindowPNG(out) // window + drawing capture the whole app (sheet/ribbon/browser)
}

// statorView is one base view to drop on the drawing sheet: an orientation index (Front=0,
// Top=1, Right=2, Iso=6) and a sheet position in millimetres on the default A3 sheet.
type statorView struct {
	orientation int
	x, y        float64
}

// buildStatorDrawing creates a drawing that documents the motor's stator part with a classic
// front / top / right / iso layout. The drawing body resolver projects a single part body, so
// the stator's toothed annulus (one body) is the natural subject for a clean engineering sheet.
func buildStatorDrawing(s *app.Session) error {
	name, ok := partDocumentName(s, "Stator")
	if !ok {
		return fmt.Errorf("no Stator part document in the workspace")
	}
	if _, err := s.NewDrawing(); err != nil {
		return err
	}
	c, err := app.ActiveDrawing(s)
	if err != nil {
		return err
	}
	c.SetModelReference(name)
	views := []statorView{{0, 135, 165}, {1, 135, 80}, {2, 285, 165}, {6, 330, 215}}
	for _, v := range views {
		if err := placeBaseView(s, v); err != nil {
			return err
		}
	}
	return nil
}

// placeBaseView drops one base view at its sheet position, failing loudly if the projection
// produced no geometry (a wrong model reference or unprojectable body).
func placeBaseView(s *app.Session, v statorView) error {
	t := app.NewBaseViewTool()
	t.Start(s)
	t.Params().Choices[0].Set(v.orientation)
	if len(t.PreviewCurves(s)) == 0 {
		return fmt.Errorf("base view orientation %d projected no curves", v.orientation)
	}
	t.SetPlacement(v.x, v.y)
	return t.Commit(s)
}

// partDocumentName returns the full document name of the first part whose display name
// contains want (e.g. "Stator"), for use as a drawing model reference.
func partDocumentName(s *app.Session, want string) (string, bool) {
	for _, d := range s.Workspace().Documents() {
		if _, ok := d.Content().(*compdef.PartComponentDefinition); !ok {
			continue
		}
		if strings.Contains(d.DisplayName(), want) {
			return d.FullDocumentName(), true
		}
	}
	return "", false
}
