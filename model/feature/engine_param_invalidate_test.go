// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/model/health"
	"oblikovati.org/model/param"
)

// paramReader is a fake feature that reads one parameter directly during recompute (modelling a
// sheet-metal thickness): the engine's read capture records it in paramReads, so MarkDirtyForParams
// can target the feature.
type paramReader struct {
	p *param.Parameter
}

func (paramReader) Kind() string { return "paramReader" }
func (f paramReader) Recompute(in Input) (Output, error) {
	_ = f.p.ModelValue() // the captured direct read
	return Output{Bodies: in.Bodies}, nil
}

// mustUserParam adds a user parameter to ps, failing the test on error.
func mustUserParam(t *testing.T, ps *param.Parameters, name, expr string) *param.Parameter {
	t.Helper()
	p, err := ps.AddUserParameter(name, expr)
	if err != nil {
		t.Fatalf("AddUserParameter(%q): %v", name, err)
	}
	return p
}

func TestMarkDirtyForParamsTargetsDirectReaderAndTail(t *testing.T) {
	ps := param.NewParameters()
	a := mustUserParam(t, ps, "a", "10 mm")
	b := mustUserParam(t, ps, "b", "20 mm")
	fs := NewPartFeatures(ps, nil)
	fs.Add(body())            // 0: independent of any parameter
	fs.Add(paramReader{p: a}) // 1: reads a
	fs.Add(body())            // 2: tail of feature 1
	fs.Recompute()

	// Editing a dirties the reader and its tail, not the clean prefix.
	fs.MarkDirtyForParams([]param.ID{a.ID()})
	fs.Recompute()
	if c := fs.Item(0).RecomputeCount(); c != 1 {
		t.Errorf("feature 0 (independent) recomputed again: count=%d, want 1", c)
	}
	if c1, c2 := fs.Item(1).RecomputeCount(), fs.Item(2).RecomputeCount(); c1 != 2 || c2 != 2 {
		t.Errorf("reader/tail not rebuilt: counts=%d/%d, want 2/2", c1, c2)
	}

	// Editing b (read by no feature) dirties nothing.
	fs.MarkDirtyForParams([]param.ID{b.ID()})
	fs.Recompute()
	for i := 0; i < fs.Count(); i++ {
		if c := fs.Item(i).RecomputeCount(); (i == 0 && c != 1) || (i > 0 && c != 2) {
			t.Errorf("feature %d recomputed on unrelated edit: count=%d", i, c)
		}
	}
}

func TestMarkDirtyForParamsAlwaysRetriesSickFeature(t *testing.T) {
	ps := param.NewParameters()
	unrelated := mustUserParam(t, ps, "x", "1 mm")
	fs := NewPartFeatures(ps, nil)
	fs.Add(body())   // 0
	fs.Add(failer{}) // 1: goes sick
	fs.Recompute()
	if fs.Item(1).Health().Status != health.Sick {
		t.Fatalf("failer health = %v, want sick", fs.Item(1).Health().Status)
	}
	before := fs.Item(1).RecomputeCount()

	// An unrelated parameter edit must still retry the sick feature (fixing a parameter can
	// recover it), even though it read no parameter.
	fs.MarkDirtyForParams([]param.ID{unrelated.ID()})
	fs.Recompute()
	if after := fs.Item(1).RecomputeCount(); after != before+1 {
		t.Errorf("sick feature not retried on parameter edit: count %d→%d", before, after)
	}
}

func TestMarkDirtyForParamsEmptyChangeIsNoOp(t *testing.T) {
	fs := NewPartFeatures(param.NewParameters(), nil)
	fs.Add(body())
	fs.Recompute()
	fs.MarkDirtyForParams(nil)
	fs.Recompute()
	if c := fs.Item(0).RecomputeCount(); c != 1 {
		t.Errorf("empty change set rebuilt a feature: count=%d, want 1", c)
	}
}
