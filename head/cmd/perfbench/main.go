//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command perfbench is the large-assembly performance driver (M34-B5): it loads (or
// generates) a synthetic automotive assembly and runs four scenarios against the real
// DrawChrome viewport — cold load & buffer upload, a 360° ViewCube orbit, a model-tree
// (browser) build stress, and parametric propagation — emitting a JSON result and a
// human summary. Memory/GC per scenario comes from perf/benchprof; set OBK_PPROF_DIR to
// also capture CPU+heap profiles.
//
//	go run ./head/cmd/perfbench -profile auto30k -out /tmp/perf.json
//	go run ./head/cmd/perfbench -assembly /tmp/car30k/asm/root_000001.oad
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/benchgen"
	"oblikovati.org/model/doc"
	"oblikovati.org/perf/benchprof"
	"oblikovati.org/persistence"
)

// Result is the full perfbench report, serialized to JSON.
type Result struct {
	Profile        string         `json:"profile"`
	Source         string         `json:"source"` // "generated" or the loaded path
	LeafPlacements int            `json:"leafPlacements"`
	UniqueMeshes   int            `json:"uniqueMeshes"`
	ColdLoad       ScenarioResult `json:"coldLoad"`
	Orbit          OrbitResult    `json:"orbit"`
	UIStress       ScenarioResult `json:"uiStress"`
	Propagation    ScenarioResult `json:"propagation"`
}

func main() {
	profileName := flag.String("profile", "auto30k", "profile to generate when -assembly is empty: auto30k|auto1m")
	assembly := flag.String("assembly", "", "root .oad to cold-load from disk (overrides -profile)")
	out := flag.String("out", "", "write the JSON result to this path (also printed to stdout)")
	png := flag.String("png", "", "save a viewport PNG after the orbit (visual confirmation)")
	frames := flag.Int("frames", 120, "frames in the 360° orbit scenario")
	uiIters := flag.Int("ui-iters", 200, "model-tree builds in the UI stress scenario")
	flag.Parse()
	if err := run(*profileName, *assembly, *out, *png, *frames, *uiIters); err != nil {
		fmt.Fprintln(os.Stderr, "perfbench:", err)
		os.Exit(1)
	}
}

func run(profileName, assembly, out, png string, frames, uiIters int) error {
	s := app.NewSessionWithStore(persistence.NewPackageStore())
	res := Result{Profile: profileName, Source: "generated"}
	cold, err := loadOrGenerate(s, profileName, assembly, &res)
	if err != nil {
		return err
	}
	res.ColdLoad = cold

	win, err := native.CreateWindow(1600, 1000, "perfbench")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	res.ColdLoad.FirstFrameMs = firstFrameUpload(win, s)

	s.FitView()
	res.Orbit = orbitScenario(win, s, frames)
	res.UIStress = uiStressScenario(s, uiIters)
	res.Propagation = propagationScenario(s)

	if png != "" {
		drawFrame(win, s)
		if err := win.SaveViewportPNG(png); err != nil {
			return fmt.Errorf("save png: %w", err)
		}
		fmt.Println("wrote", png)
	}
	return report(res, out)
}

// loadOrGenerate fills the session with the assembly — opening the saved root (the true
// cold load: disk read + reference resolve + recompute) or generating it in memory —
// and returns the cold-load timing/memory. It records placement and unique-mesh counts
// from the flattened scene.
func loadOrGenerate(s *app.Session, profileName, assembly string, res *Result) (ScenarioResult, error) {
	prof, err := benchprof.Start("coldload")
	if err != nil {
		return ScenarioResult{}, err
	}
	t0 := time.Now()
	root, genErr := fillSession(s, profileName, assembly, res)
	if genErr != nil {
		return ScenarioResult{}, genErr
	}
	if err := s.Workspace().SetActiveDocument(root); err != nil {
		return ScenarioResult{}, err
	}
	groups := s.VisibleInstances()
	res.LeafPlacements = totalTransforms(groups)
	res.UniqueMeshes = len(groups)
	return finishScenario(prof, t0)
}

// fillSession opens the assembly from disk when assembly is set, else generates the
// named profile in memory, returning the root document.
func fillSession(s *app.Session, profileName, assembly string, res *Result) (*doc.Document, error) {
	if assembly != "" {
		res.Source = assembly
		return s.Workspace().Open(assembly, true)
	}
	profile, err := benchgen.ProfileByName(profileName)
	if err != nil {
		return nil, err
	}
	root, _, err := benchgen.Generate(s.Workspace(), "perfbench", profile)
	return root, err
}

// firstFrameUpload times the first rendered frame — the cold vertex/index/instance
// buffer upload to the GPU — in milliseconds.
func firstFrameUpload(win *native.Window, s *app.Session) float64 {
	t0 := time.Now()
	drawFrame(win, s)
	return msSince(t0)
}

// report marshals the result to JSON, prints the human summary to stdout, and optionally
// writes the JSON to a file.
func report(res Result, out string) error {
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	printSummary(res)
	if out == "" {
		return nil
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %q: %w", out, err)
	}
	fmt.Println("wrote", out)
	return nil
}
