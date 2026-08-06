// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/test-utilities/degenerate"

	"oblikovati.org/app"
)

// crossedTrimFeature is a feature whose result is deliberately BAD INPUT for the tessellator — a
// self-crossing trim boundary no triangulation can cover (test-utilities/degenerate). It is a fake
// on purpose: a body the kernel builds and then meshes badly is a tessellation bug to fix, so there
// is no shipping feature that reliably degrades, and enshrining one as a fixture would mean shipping
// the bug it depends on.
type crossedTrimFeature struct{}

func (crossedTrimFeature) Kind() string { return "crossed-trim" }

func (crossedTrimFeature) Recompute(feature.Input) (feature.Output, error) {
	return feature.Output{Bodies: []*topo.Body{degenerate.CrossedTrimBody()}}, nil
}

// TestFeatureReplyCarriesTessellationDiagnostics is the #2058 wire-level regression: the reply an
// add-in reads must carry what the tessellator recorded, keyed on the tessellation code. This is the
// assertion #2038 would have failed — there the rim wall meshed to half its area, the body's volume
// came back 77% low, and `diagnostics` was an empty array.
func TestFeatureReplyCarriesTessellationDiagnostics(t *testing.T) {
	def := emptyPart(t)
	pf := def.Features().Add(crossedTrimFeature{})
	out, err := recomputeResult(def, pf)
	if err != nil {
		t.Fatalf("recomputeResult: %v", err)
	}
	var r struct {
		Diagnostics []struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
			Detail   string `json:"detail"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("unmarshal reply: %v", err)
	}
	for _, d := range r.Diagnostics {
		if d.Code == string(ops.CodePatchCoverage) && d.Severity == "defect" {
			return
		}
	}
	t.Errorf("feature reply carries no %q defect; got %s", ops.CodePatchCoverage, out)
}

// emptyPart seeds a session with one empty part document.
func emptyPart(t *testing.T) *compdef.PartComponentDefinition {
	t.Helper()
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "tessdiag.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	return def
}
