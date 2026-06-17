// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// insertB1DXF defines a block B1 holding a unit LINE (0,0)→(1,0) and inserts it at (10,0)
// scaled ×2 and rotated 90°. Expansion should place the line at (10,0)→(10,2).
const insertB1DXF = `0
SECTION
2
BLOCKS
0
BLOCK
2
B1
70
0
10
0.0
20
0.0
3
B1
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
ENDBLK
0
ENDSEC
0
SECTION
2
ENTITIES
0
INSERT
2
B1
10
10.0
20
0.0
41
2.0
42
2.0
43
1.0
50
90
0
ENDSEC
0
EOF
`

// TestInsertExpansion checks a model-space INSERT expands into transformed copies of its
// block's geometry (scale, rotation, translation applied).
func TestInsertExpansion(t *testing.T) {
	dr, warns, err := Decode([]byte(insertB1DXF))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("warnings: %v", warns)
	}
	if len(dr.Entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(dr.Entities))
	}
	l := dr.Entities[0].(*drawing.Line)
	if !near(l.Start[0], 10) || !near(l.Start[1], 0) || !near(l.End[0], 10) || !near(l.End[1], 2) {
		t.Errorf("expanded line = %v→%v, want (10,0)→(10,2)", l.Start, l.End)
	}
}

// nestedDXF: block OUTER inserts block INNER (a unit line); the model inserts OUTER at (5,0).
const nestedDXF = `0
SECTION
2
BLOCKS
0
BLOCK
2
INNER
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
ENDBLK
0
BLOCK
2
OUTER
0
INSERT
2
INNER
10
0.0
20
0.0
0
ENDBLK
0
ENDSEC
0
SECTION
2
ENTITIES
0
INSERT
2
OUTER
10
5.0
20
0.0
0
ENDSEC
0
EOF
`

// TestNestedInsertExpansion checks an INSERT of a block that itself contains an INSERT
// expands recursively, composing the transforms.
func TestNestedInsertExpansion(t *testing.T) {
	dr, _, err := Decode([]byte(nestedDXF))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(dr.Entities) != 1 {
		t.Fatalf("entities = %d, want 1", len(dr.Entities))
	}
	l := dr.Entities[0].(*drawing.Line)
	if !near(l.Start[0], 5) || !near(l.End[0], 6) {
		t.Errorf("nested line = %v→%v, want (5,0)→(6,0)", l.Start, l.End)
	}
}

// cyclicDXF: block A holds a line and inserts B; block B inserts A — a reference cycle. The
// expander must terminate.
const cyclicDXF = `0
SECTION
2
BLOCKS
0
BLOCK
2
A
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
INSERT
2
B
10
0.0
20
0.0
0
ENDBLK
0
BLOCK
2
B
0
INSERT
2
A
10
0.0
20
0.0
0
ENDBLK
0
ENDSEC
0
SECTION
2
ENTITIES
0
INSERT
2
A
10
0.0
20
0.0
0
ENDSEC
0
EOF
`

// TestCyclicInsertTerminates checks a block reference cycle does not loop forever and yields
// a finite result (at least the line from block A).
func TestCyclicInsertTerminates(t *testing.T) {
	dr, _, err := Decode([]byte(cyclicDXF))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(dr.Entities) < 1 {
		t.Fatalf("entities = %d, want >= 1", len(dr.Entities))
	}
}

// TestPaperSpaceSkipped checks an entity flagged paper space (code 67 = 1) is not imported.
func TestPaperSpaceSkipped(t *testing.T) {
	const dxf = `0
SECTION
2
ENTITIES
0
LINE
67
1
10
0.0
20
0.0
11
1.0
21
1.0
0
LINE
10
0.0
20
0.0
11
2.0
21
2.0
0
ENDSEC
0
EOF
`
	dr, _, err := Decode([]byte(dxf))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(dr.Entities) != 1 {
		t.Fatalf("entities = %d, want 1 (paper-space line skipped)", len(dr.Entities))
	}
	l := dr.Entities[0].(*drawing.Line)
	if !near(l.End[0], 2) {
		t.Errorf("kept the wrong line: %v→%v", l.Start, l.End)
	}
}
