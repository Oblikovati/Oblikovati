// SPDX-License-Identifier: GPL-2.0-only

package examples_test

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"oblikovati/addin/opregistry"
	"oblikovati/addin/router"
	"oblikovati/app"
	"oblikovati/script"
	"oblikovati/script/bridge"
	"oblikovati/script/examples"
	"oblikovati/script/gopherlua"
	"oblikovati/script/runner"
)

// runExample runs a bundled example against a real Session + router (the in-proc CLI
// wiring) and returns the router + session so a test can query the resulting model
// independently of whatever the script itself printed.
func runExample(t *testing.T, name string) (*router.Router, *app.Session) {
	t.Helper()
	s := app.NewSession()
	rtr := router.New(opregistry.Default())
	caller := bridge.NewDirectCaller(rtr.Handle, s)
	r := runner.New(gopherlua.New(), caller, rtr.Methods)
	src, err := examples.Source(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	res, err := r.Run(context.Background(), src, script.Limits{Wall: 10 * time.Second}, nil)
	if err != nil {
		t.Fatalf("run %s: %v", name, err)
	}
	if res.Err != nil {
		t.Fatalf("%s script error: %v", name, res.Err)
	}
	return rtr, s
}

// query invokes a wire method on the same session and decodes the JSON result into v.
func query(t *testing.T, rtr *router.Router, s *app.Session, method, args string, v any) {
	t.Helper()
	out, err := rtr.Handle(s, method, []byte(args))
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	if v != nil {
		if err := json.Unmarshal(out, v); err != nil {
			t.Fatalf("%s: decode: %v", method, err)
		}
	}
}

// TestAllExamplesRun is the corpus guard: every bundled example must run to completion
// against the real model without a script error, so a contract change that breaks an
// example is caught here.
func TestAllExamplesRun(t *testing.T) {
	names, err := examples.Names()
	if err != nil {
		t.Fatalf("list examples: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no bundled examples found")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) { runExample(t, name) })
	}
}

type paramList struct {
	Parameters []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"parameters"`
}

func paramValue(pl paramList, name string) (string, bool) {
	for _, p := range pl.Parameters {
		if p.Name == name {
			return p.Value, true
		}
	}
	return "", false
}

func TestExampleCreateParameters(t *testing.T) {
	rtr, s := runExample(t, "create_parameters.lua")
	var pl paramList
	query(t, rtr, s, "parameters.list", "{}", &pl)
	if v, ok := paramValue(pl, "Width"); !ok || !strings.Contains(v, "76.2") {
		t.Errorf("Width = %q (ok=%v), want ~76.2 mm (3 in)", v, ok)
	}
	if v, ok := paramValue(pl, "Height"); !ok || !strings.Contains(v, "40") {
		t.Errorf("Height = %q (ok=%v), want 40 mm", v, ok)
	}
}

func TestExampleSetParameter(t *testing.T) {
	rtr, s := runExample(t, "set_parameter.lua")
	var p struct {
		Value string `json:"value"`
	}
	query(t, rtr, s, "parameters.get", `{"name":"Length"}`, &p)
	if !strings.Contains(p.Value, "88.9") { // 3.5 in
		t.Errorf("Length = %q, want ~88.9 mm (3.5 in)", p.Value)
	}
}

func TestExampleSketchLines(t *testing.T) {
	rtr, s := runExample(t, "sketch_lines.lua")
	var ents struct {
		Entities []struct {
			Kind string `json:"kind"`
		} `json:"entities"`
	}
	query(t, rtr, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	lines := 0
	for _, e := range ents.Entities {
		if e.Kind == "line" {
			lines++
		}
	}
	if lines != 7 { // 3 (triangle) + 4 (rectangle)
		t.Errorf("line count = %d, want 7", lines)
	}
}

// physProps is the model.physicalProperties result shape (api/types.PhysicalProperties).
type physProps struct {
	Volume   float64    `json:"volume"`
	Area     float64    `json:"area"`
	Centroid [3]float64 `json:"centroid"`
}

func TestExampleExtrudeBlock(t *testing.T) {
	rtr, s := runExample(t, "extrude_block.lua")
	var p physProps
	query(t, rtr, s, "model.physicalProperties", "{}", &p)
	if math.Abs(p.Volume-12) > 0.05 { // 4*3*1 cm
		t.Errorf("block volume = %.4f cm3, want 12", p.Volume)
	}
}

func TestExampleRevolveTube(t *testing.T) {
	rtr, s := runExample(t, "revolve_tube.lua")
	var p physProps
	query(t, rtr, s, "model.physicalProperties", "{}", &p)
	want := 16 * math.Pi // pi*(3^2-1^2)*2
	if rel := math.Abs(p.Volume-want) / want; rel > 0.03 {
		t.Errorf("tube volume = %.4f cm3, want ~%.4f (rel err %.3f)", p.Volume, want, rel)
	}
}

func TestExampleMassProperties(t *testing.T) {
	rtr, s := runExample(t, "mass_properties.lua")
	var p physProps
	query(t, rtr, s, "model.physicalProperties", "{}", &p)
	if math.Abs(p.Volume-8) > 0.05 { // 2 cm cube
		t.Errorf("cube volume = %.4f cm3, want 8", p.Volume)
	}
	if math.Abs(p.Area-24) > 0.1 { // 6 faces * 4 cm2
		t.Errorf("cube area = %.4f cm2, want 24", p.Area)
	}
	for i, c := range p.Centroid {
		if math.Abs(c-1) > 0.05 { // centred at (1,1,1) cm
			t.Errorf("centroid[%d] = %.4f, want ~1", i, c)
		}
	}
}
