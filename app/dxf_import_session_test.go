// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// twoLineDXF is a minimal ASCII DXF with two model-space lines.
const twoLineDXF = `0
SECTION
2
ENTITIES
0
LINE
10
0.0
20
0.0
11
1.0
21
0.0
0
LINE
10
0.0
20
0.0
11
0.0
21
1.0
0
ENDSEC
0
EOF
`

// TestSessionImportDXF imports a small .dxf onto the default plane and checks it lands as a
// populated 2D sketch (an undoable edit), exercising the session→model→codec import path.
func TestSessionImportDXF(t *testing.T) {
	s := sessionWithPart(t)
	path := filepath.Join(t.TempDir(), "in.dxf")
	if err := os.WriteFile(path, []byte(twoLineDXF), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := s.ImportDXFOnPlane(path, "")
	if err != nil {
		t.Fatalf("ImportDXFOnPlane: %v", err)
	}
	if res.Is3D {
		t.Errorf("planar drawing imported as 3D")
	}
	if res.EntityCount != 2 {
		t.Errorf("entity count = %d, want 2", res.EntityCount)
	}
}

// TestSessionExportActiveSketchDXF builds an active sketch, exports it to .dxf via the
// session and re-imports it, checking the geometry round-trips through the surfaced path.
func TestSessionExportActiveSketchDXF(t *testing.T) {
	s := sessionWithPart(t)
	part, err := activePart(s)
	if err != nil {
		t.Fatal(err)
	}
	sk := part.Sketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(2, 3))
	if err := s.EditSketch(sk); err != nil {
		t.Fatalf("EditSketch: %v", err)
	}
	path := filepath.Join(t.TempDir(), "out.dxf")
	n, err := s.ExportActiveSketchDXF(path, types.DXFR2000)
	if err != nil {
		t.Fatalf("ExportActiveSketchDXF: %v", err)
	}
	if n != 1 {
		t.Errorf("exported %d curves, want 1", n)
	}
	res, err := s.ImportDXFOnPlane(path, "")
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if res.EntityCount != 1 {
		t.Errorf("re-imported %d entities, want 1", res.EntityCount)
	}
}

// TestSessionExportDXFNoActiveSketch errors clearly when there is no active sketch.
func TestSessionExportDXFNoActiveSketch(t *testing.T) {
	s := sessionWithPart(t)
	if _, err := s.ExportActiveSketchDXF(filepath.Join(t.TempDir(), "x.dxf"), types.DXFR2000); err == nil {
		t.Error("expected an error with no active sketch")
	}
}
