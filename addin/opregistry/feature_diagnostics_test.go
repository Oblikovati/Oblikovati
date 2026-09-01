// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// circledPart seeds a part with two circle sketches (radius 2 at origin, radius 1 at x=1.5) so
// an extrude+cut pair forms two crossing analytic cylinders — the configuration no exact curved
// path handles, which facets the operands for the planar boolean.
func circledPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "diag.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	def.Sketches().Add(sketch.XYPlane()).Circles().AddByCenterRadius(math.P2(0, 0), 2)
	def.Sketches().Add(sketch.XYPlane()).Circles().AddByCenterRadius(math.P2(1.5, 0), 1)
	def.Recompute()
	return s
}

// TestFeatureReplyCarriesFallbackDiagnostics is the #1601 wire-level regression: a cut that
// facets its analytic operands (cylinder on cylinder, no exact curved path) must report the
// degradation in the feature reply's diagnostics — the API caller's only window into it.
func TestFeatureReplyCarriesFallbackDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	s := circledPart(t)
	if _, err := apply(t, s, "extrude", `{"sketchIndex":0,"distance":"30 mm","operation":"new"}`); err != nil {
		t.Fatalf("base cylinder: %v", err)
	}
	out, err := apply(t, s, "extrude", `{"sketchIndex":1,"distance":"30 mm","operation":"cut"}`)
	if err != nil {
		t.Fatalf("cylinder-on-cylinder cut: %v", err)
	}
	var r struct {
		Healthy     bool `json:"healthy"`
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Detail   string `json:"detail"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("unmarshal reply: %v", err)
	}
	if !r.Healthy {
		t.Fatalf("cut reported unhealthy; want healthy-but-degraded: %s", out)
	}
	for _, d := range r.Diagnostics {
		if d.Code == string(ops.CodeBooleanAnalyticFaceted) {
			return
		}
	}
	t.Errorf("feature reply carries no %q diagnostic; got %s", ops.CodeBooleanAnalyticFaceted, out)
}
