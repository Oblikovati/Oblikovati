// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/model/param"
)

// TestSketchesShareParametersWiresNewSketch covers the fix: a collection given a shared
// parameter DAG hands it to every sketch it creates (so dimensions resolve), and a bare
// collection leaves a sketch with its own set.
func TestSketchesShareParametersWiresNewSketch(t *testing.T) {
	ps := param.NewParameters()
	shared := NewSketches()
	shared.ShareParameters(ps)
	if got := shared.Add(XYPlane()).Parameters(); got != ps {
		t.Errorf("shared collection: sketch params = %p, want the shared set %p", got, ps)
	}

	bare := NewSketches()
	if got := bare.Add(XYPlane()).Parameters(); got == nil || got == ps {
		t.Error("bare collection: sketch should keep its own non-shared parameter set")
	}
}

// TestSketches3DShareParametersWiresNewSketch is the 3D counterpart.
func TestSketches3DShareParametersWiresNewSketch(t *testing.T) {
	ps := param.NewParameters()
	shared := NewSketches3D()
	shared.ShareParameters(ps)
	if got := shared.Add().Parameters(); got != ps {
		t.Errorf("shared 3D collection: sketch params = %p, want the shared set %p", got, ps)
	}

	bare := NewSketches3D()
	if got := bare.Add().Parameters(); got == nil || got == ps {
		t.Error("bare 3D collection: sketch should keep its own non-shared parameter set")
	}
}
