# OCCT Blend Parity Test Corpus — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port OpenCASCADE's `tests/blend/*` corpus (477 cases) into hard-asserting Go
tests that drive the real `FilletFeature` path and are gated at OCCT's own 1% area
tolerance, producing a live parity scoreboard.

**Architecture:** An **OCCT oracle** (a DRAWEXE-class tool linked against the already-built
OCCT libs) runs each `.tcl` case once and emits, per case, a JSON record + a STEP export of
OCCT's exact input solid: the blend picks (radius + a **geometric edge locator** = midpoint
& direction, OCCT's own edge resolution), any variable-radius law, and OCCT's reference
surface area. A Go generator turns those records into a `corpus.go` table + committed STEP
fixtures. The `occtparity` test harness imports each case's STEP input, **geometrically
locates** the picked edges on our body, runs our fillet feature, and asserts area within
OCCT's tolerance. The harness never re-implements OCCT's shape DSL — the oracle supplies
identical input geometry as STEP, so the only variable under test is *our fillet*.

**Tech Stack:** Go (Oblikovati kernel/model), C++/CMake (the one-time oracle, linked to
`occt-install`), OCCT DRAW/Tcl, STEP interchange (`kernel/exchange/step`).

**Refines the spec** (`docs/superpowers/specs/2026-07-11-occt-blend-parity-corpus-design.md`):
the spec proposed a Go `draw.go`/`profile.go` DSL that reproduces OCCT's `box`/`prism`/…
construction and computes edge locators from primitive definitions. Planning grounding
showed that (a) 292/477 cases load external `.rle`/`.brep` fixtures the DSL cannot build,
and (b) reproducing OCCT's `explode` edge order by hand is brittle and impossible for
restored fixtures. The oracle-STEP approach subsumes both: OCCT itself exports the input and
resolves the edges. `draw.go`/`profile.go` are therefore **not built**; `checkprops.go`,
`edgepick.go`, `status.go` remain as specified.

## Global Constraints

- **Assertion is surface AREA only.** OCCT `checkprops` uses `-s` in 475/475 blend cases;
  `-v` never appears. Volume is computed as a non-gating manifold sanity check only.
- **Tolerance = OCCT's own.** Gate: `abs((expected-actual)/expected) ≤ deps`, `deps = 0.01`
  (1%) unless the case's `.tcl` overrides with `-deps`. No blend case overrides, so 1% for
  all. Additionally emit a non-failing `t.Logf` warning when drift `> 0.001` (0.1%).
- **Mirror OCCT's own outcome.** A case whose `.tcl` carries `puts "TODO … TEST
  INCOMPLETE"` (16 cases, correlated with `checkprops … -s 0`) is ported as
  expected-incomplete (`t.Skip` with the TODO string) — we are never stricter than OCCT.
- **Validity mirrors `parse.rules`.** A result that is not a valid closed solid (our
  `Validate` + non-zero volume) fails — the analogue of OCCT's `\bFaulty\b` FAILED rule.
- **Drive the real feature path.** Fillets go through `NewDressUpFeatures(fs).AddFillet…`
  + `fs.Recompute()`, never raw `ops.FilletEdges` (per `validate-fillet-through-feature`).
- **Harness is test-scope, at the feature layer.** Package `model/feature/occtparity`,
  imported only by `_test.go`; imports `model/feature`, `kernel/exchange/step`,
  `kernel/ops`, `kernel/topo`, `oblikovati.org/math`. Never ships.
- **No PR until the entire ported corpus is green** (excluding OCCT-TODO cases). Long-lived
  branch `feat/occt-blend-parity-corpus`, granular commits per task. Engine "greening" work
  (turning red cases green) is tracked separately, out of this plan's scope.
- **SPDX header** `// SPDX-License-Identifier: GPL-2.0-only` on every new `.go` file.
- **math import** is `oblikovati.org/math` (`math.P2/P3/V2/V3`), not stdlib; alias stdlib
  as `stdmath "math"` where both are needed.

## Grounded API reference (verified — use verbatim)

- Base body: `fs := feature.NewPartFeatures(nil)`; `feature.NewBaseFeatures(fs).AddBase(body *topo.Body)`.
- Fillet (constant): `feature.NewDressUpFeatures(fs).AddFillet(edgeKeys [][]byte, radius func() float64) *PartFeature`.
- Fillet (per-set radii): `.AddFilletSets([]feature.FilletEdgeSet{{EdgeKeys: [][]byte, Radius: func() float64}})`.
- Fillet (corner type): `.AddFilletCorner(edgeKeys [][]byte, radius func() float64, corner feature.FilletCornerType)`.
- Recompute/result/health: `fs.Recompute()`; `fs.Result() []*topo.Body`; `pf.Health().OK()`, `pf.Health().Reason` (string).
- Edge key: `edge.ReferenceKey() []byte`; body edges: `body.Edges() []*topo.Edge`.
- Edge locator: `topo.DescribeEdge(e) topo.GeometricEdgeRef{Midpoint math.Point3; Direction math.Vector3}`.
- Mass props: `ops.BodyGeometryProperties(b *topo.Body, ops.PropertyQuality()) ops.GeometryProperties{Volume, Area, Centroid}` (use `PropertyQuality()`, ChordTolerance 1e-3, for curved bodies).
- STEP import: `import stepio "oblikovati.org/kernel/exchange/step"` + `import "oblikovati.org/kernel/exchange"`; `stepio.Reader{}.ImportSolids(data []byte, exchange.TranslationOptions{TargetUnitMM: 1}) ([]*topo.Body, []string, error)`.
- Variable radius (kernel): `ops.EdgeFilletRadii{Key []byte; R0, R1 float64; Mids []ops.FilletRadiusPoint}` via `ops.FilletEdgesVarying(body, picks)` — used only where the feature layer lacks a variable-radius Add (see Task 12).

---

## Phase 0 — The OCCT oracle

### Task 1: Build a DRAWEXE-class oracle against the existing OCCT libs

**Files:**
- Create: `test-utilities/occt-blend/oracle/CMakeLists.txt`
- Create: `test-utilities/occt-blend/oracle/occt_blend_oracle.cxx`
- Create: `test-utilities/occt-blend/oracle/README.md`

**Interfaces:**
- Produces: a binary `occt_blend_oracle` that, given a `.tcl` case path + an output dir,
  writes `<out>/<case>.step` (OCCT input solid) and `<out>/<case>.json` (see Task 3 schema).
  For this task it need only handle the **inline-primitive constant-`blend`** form
  (`box`/`pcylinder`/… + `explode` + `blend result s r e …` + `checkprops -s`).

The oracle links the already-built toolkits in
`/home/vmiguel/git/oblikovati-workspace/occt-install/lib` (TKernel, TKMath, TKBRep,
TKTopAlgo, TKFillet, TKGeomAlgo, TKMesh, TKPrim, TKSTEP, TKG3d, TKShHealing). It embeds a
Tcl interpreter (`libtcl8.6`, present) with the DRAW modeling commands so real `.tcl`
scripts run unmodified. If enabling OCCT's own `TKDraw`/`DRAWEXE` from the existing build
tree (`occt-build`, reconfigure `-DBUILD_MODULE_Draw=ON`) is faster than a bespoke embed,
that is an acceptable substitute **provided** it can still emit the per-case JSON+STEP (via
a sourced Tcl helper that calls `stepwrite` + a custom `dumpblend` command). Decide by
spike; record the choice in `README.md`.

- [ ] **Step 1: Write the failing check — the oracle binary does not exist yet**

Run: `test -x test-utilities/occt-blend/oracle/build/occt_blend_oracle && echo FOUND || echo MISSING`
Expected: `MISSING`

- [ ] **Step 2: Author `CMakeLists.txt` linking occt-install**

```cmake
cmake_minimum_required(VERSION 3.16)
project(occt_blend_oracle CXX)
set(CMAKE_CXX_STANDARD 17)
set(OCCT_ROOT /home/vmiguel/git/oblikovati-workspace/occt-install)
find_package(Tcl REQUIRED)
add_executable(occt_blend_oracle occt_blend_oracle.cxx)
target_include_directories(occt_blend_oracle PRIVATE ${OCCT_ROOT}/include/opencascade ${TCL_INCLUDE_PATH})
target_link_directories(occt_blend_oracle PRIVATE ${OCCT_ROOT}/lib)
target_link_libraries(occt_blend_oracle PRIVATE
  TKernel TKMath TKG2d TKG3d TKGeomBase TKBRep TKGeomAlgo TKTopAlgo TKPrim
  TKShHealing TKFillet TKBO TKBool TKMesh TKSTEP TKSTEPBase TKXSBase ${TCL_LIBRARY})
```

- [ ] **Step 3: Implement the oracle for the inline-primitive constant-blend form**

`occt_blend_oracle.cxx` (skeleton — the spike fills DRAW command wiring):
```cpp
// SPDX-License-Identifier: GPL-2.0-only
// Runs one OCCT tests/blend .tcl case and emits <case>.step (input solid) + <case>.json.
#include <tcl.h>
#include <BRepFilletAPI_MakeFillet.hxx>
#include <STEPControl_Writer.hxx>
#include <GProp_GProps.hxx>
#include <BRepGProp.hxx>
#include <TopExp.hxx>
#include <TopTools_IndexedMapOfShape.hxx>
// ... a custom Tcl command `oracle_dump` reads the DRAW variable `s` (input) and the
// blend spec captured by intercepting `blend`; writes STEP of `s`, then for each picked
// edge computes midpoint+tangent (BRepAdaptor_Curve at mid param) and the fillet radius,
// and records checkprops' -s value. See README for the interception approach.
int main(int argc, char** argv){ /* init interp, load DRAW cmds, source case, dump */ }
```

The load-bearing detail (document in README): intercept OCCT's `blend`/`mkevol`/`updatevol`
DRAW commands with Tcl wrappers that record `(radius, edgeName)` / `(edge, law)` before
delegating to the real command, then after the script runs, resolve each recorded edge name
to its shape via the interp's DRAW variable and compute its midpoint+direction with
`BRepAdaptor_Curve`. This yields OCCT's *own* edge resolution as a geometry-independent
locator.

- [ ] **Step 4: Build the oracle**

Run: `cmake -S test-utilities/occt-blend/oracle -B test-utilities/occt-blend/oracle/build && cmake --build test-utilities/occt-blend/oracle/build -j`
Expected: `occt_blend_oracle` binary produced.

- [ ] **Step 5: Smoke-run on `blend/simple/A1` and verify STEP + JSON emitted**

Run: `LD_LIBRARY_PATH=$PWD/../occt-install/lib test-utilities/occt-blend/oracle/build/occt_blend_oracle ../OCCT/tests/blend/simple/A1 /tmp/oracle-out && ls /tmp/oracle-out/A1.step /tmp/oracle-out/A1.json`
Expected: both files exist; `A1.json` contains one pick with radius 10 and a midpoint on a
100³ box edge; `A1.step` imports (checked in Task 4).

- [ ] **Step 6: Commit**

```bash
git add test-utilities/occt-blend/oracle
git commit -m "test(occt-parity): OCCT blend oracle — export input STEP + blend-pick locators per case"
```

### Task 2: Fetch OCCT test-data fixtures for the restore-cases

**Files:**
- Create: `test-utilities/occt-blend/SOURCES.md`
- Create (script): `test-utilities/occt-blend/fetch-data.sh`

**Interfaces:**
- Produces: a local checkout of OCCT's test-data repo and a resolver so the oracle's
  `locate_data_file` finds the ~170 referenced fixtures. Only the referenced files are
  retained under `test-utilities/occt-blend/data/` (not the whole repo).

- [ ] **Step 1: List the distinct data files the corpus references**

Run: `grep -rhoE 'locate_data_file [^]]+' ../OCCT/tests/blend/*/[A-Z]* | sed 's/locate_data_file //' | sort -u | tee /tmp/needed-fixtures.txt | wc -l`
Expected: ~170 names printed.

- [ ] **Step 2: Write `fetch-data.sh` to clone OCCT test-data and copy only needed files**

```bash
#!/usr/bin/env bash
# Clones OCCT's public test-data repo and copies only the fixtures the blend corpus needs.
set -euo pipefail
DEST=test-utilities/occt-blend/data
SRC=${1:-/tmp/occt-test-data}
[ -d "$SRC" ] || git clone --depth 1 https://github.com/Open-Cascade-SAS/occt-test-data.git "$SRC"
mkdir -p "$DEST"
while read -r f; do find "$SRC" -name "$f" -exec cp {} "$DEST/" \; ; done < /tmp/needed-fixtures.txt
ls "$DEST" | wc -l
```

- [ ] **Step 3: Run it; record how many fixtures resolved**

Run: `bash test-utilities/occt-blend/fetch-data.sh`
Expected: a count near ~170; note in `SOURCES.md` any names that did not resolve (their
cases become import-divergence-skipped in Task 11, tracked, not silently dropped).

- [ ] **Step 4: Verify the oracle resolves a restore-case (`blend/complex/A1`)**

Run: `OCCT_DATA=test-utilities/occt-blend/data test-utilities/occt-blend/oracle/build/occt_blend_oracle ../OCCT/tests/blend/complex/A1 /tmp/oracle-out && ls /tmp/oracle-out/A1.json`
Expected: JSON emitted; since complex/A1 is an OCCT-TODO `-s 0` case, its JSON carries the
TODO marker + `expectedArea: 0`.

- [ ] **Step 5: Commit** (data files + SOURCES.md with the OCCT test-data commit SHA)

```bash
git add test-utilities/occt-blend/data test-utilities/occt-blend/fetch-data.sh test-utilities/occt-blend/SOURCES.md
git commit -m "test(occt-parity): vendor the ~170 OCCT blend test-data fixtures + provenance"
```

### Task 3: Freeze the oracle JSON schema + extend it to evol/bfuse/todo

**Files:**
- Modify: `test-utilities/occt-blend/oracle/occt_blend_oracle.cxx`
- Create: `test-utilities/occt-blend/oracle/schema.json` (documentation of the record shape)

**Interfaces:**
- Produces: the per-case JSON contract every downstream Go task consumes:

```json
{
  "grid": "simple", "case": "A1",
  "verb": "blend",                         // "blend" | "buildevol" | "bfuseblend"
  "expectedArea": 59527.9,
  "deps": 0.01,                            // from -deps if present, else 0.01
  "todo": "",                              // OCCT TODO/INCOMPLETE marker string, else ""
  "inputStep": "A1.step",                  // OCCT input solid (for bfuseblend, the pre-fuse operands is documented below)
  "picks": [
    { "radius": 10.0,
      "locator": {"midpoint": [x,y,z], "direction": [dx,dy,dz]},
      "law": null }                        // null for constant radius
  ]
}
```
For `buildevol`, `verb:"buildevol"` and each pick's `law` is `[[p0,r0],[p1,r1],…]` (the
`updatevol` parameter→radius pairs). For `bfuseblend`, `verb:"bfuseblend"`, `inputStep` is
the **fused** solid (the oracle performs the fuse so our side blends the same edges),
`picks` carries the single radius applied to the fused seam edges (one pick per resolved
seam edge, all same radius).

- [ ] **Step 1: Write the failing schema test** (a Go test asserting a golden JSON parses)

Create `model/feature/occtparity/record_test.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import "testing"

func TestParseOracleRecordGolden(t *testing.T) {
	r, err := parseRecord([]byte(`{"grid":"simple","case":"A1","verb":"blend","expectedArea":59527.9,"deps":0.01,"todo":"","inputStep":"A1.step","picks":[{"radius":10,"locator":{"midpoint":[50,0,50],"direction":[0,0,1]},"law":null}]}`))
	if err != nil { t.Fatalf("parse: %v", err) }
	if r.Grid != "simple" || len(r.Picks) != 1 || r.Picks[0].Radius != 10 { t.Fatalf("bad record: %+v", r) }
	if r.Picks[0].Locator.Midpoint != [3]float64{50, 0, 50} { t.Fatalf("mid: %v", r.Picks[0].Locator.Midpoint) }
}
```

- [ ] **Step 2: Run it — fails (parseRecord/Record undefined)**

Run: `cd /home/vmiguel/git/oblikovati-workspace/Oblikovati && go test ./model/feature/occtparity/ -run TestParseOracleRecordGolden`
Expected: FAIL to compile — `undefined: parseRecord`.

- [ ] **Step 3: Implement `record.go`**

Create `model/feature/occtparity/record.go`:
```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import "encoding/json"

type Locator struct {
	Midpoint  [3]float64 `json:"midpoint"`
	Direction [3]float64 `json:"direction"`
}
type Pick struct {
	Radius  float64     `json:"radius"`
	Locator Locator     `json:"locator"`
	Law     [][2]float64 `json:"law"` // nil ⇒ constant radius
}
type Record struct {
	Grid, Case, Verb string  `json:"-"`
	ExpectedArea     float64 `json:"expectedArea"`
	Deps             float64 `json:"deps"`
	TODO             string  `json:"todo"`
	InputStep        string  `json:"inputStep"`
	Picks            []Pick  `json:"picks"`
}

func parseRecord(b []byte) (Record, error) {
	var r Record
	// Grid/Case/Verb carry json tags "grid"/"case"/"verb"; re-tag via an alias to keep them.
	type alias Record
	var a struct{ alias; G, C, V string `json:"grid";"case";"verb"` }
	_ = a
	if err := json.Unmarshal(b, &r); err != nil { return Record{}, err }
	var meta struct{ Grid, Case, Verb string }
	_ = json.Unmarshal(b, &meta)
	r.Grid, r.Case, r.Verb = meta.Grid, meta.Case, meta.Verb
	return r, nil
}
```
(If the struct-tag gymnastics offend, split `Record` into an exported struct with plain
`json:"grid"` tags on Grid/Case/Verb and drop the `json:"-"`. The test only cares that
Grid/Case/Verb/Picks populate.)

- [ ] **Step 4: Run it — passes**

Run: `go test ./model/feature/occtparity/ -run TestParseOracleRecordGolden`
Expected: PASS.

- [ ] **Step 5: Extend the C++ oracle to emit `buildevol`, `bfuseblend`, and `todo`**, then re-dump a buildevol case and eyeball the `law`.

Run: `test-utilities/occt-blend/oracle/build/occt_blend_oracle ../OCCT/tests/blend/buildevol/A1 /tmp/oracle-out && grep -o '"law":\[\[.*\]\]' /tmp/oracle-out/A1.json`
Expected: `"law":[[0,2],[1,4],[2,2]]` (matches `updatevol s_5 0 2 1 4 2 2`).

- [ ] **Step 6: Commit**

```bash
git add model/feature/occtparity/record.go model/feature/occtparity/record_test.go test-utilities/occt-blend/oracle
git commit -m "test(occt-parity): freeze oracle JSON record (blend/buildevol/bfuseblend/todo) + Go parser"
```

---

## Phase 1 — Corpus extraction

### Task 4: STEP round-trip fidelity pre-check

**Files:**
- Create: `model/feature/occtparity/importinput.go`
- Create: `model/feature/occtparity/importinput_test.go`

**Interfaces:**
- Produces: `func importInput(stepPath string) (*topo.Body, error)` and
  `func inputArea(b *topo.Body) float64` used by the runner to (a) load OCCT's input solid
  and (b) verify it survived STEP import before we blame the fillet.

- [ ] **Step 1: Write the failing test** — import `A1.step` (a 100³ box) and check its area ≈ 60000.

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import ( "path/filepath"; "testing" )

func TestImportInputBoxArea(t *testing.T) {
	b, err := importInput(filepath.Join("testdata", "A1.step"))
	if err != nil { t.Fatalf("import: %v", err) }
	got := inputArea(b)
	if got < 59000 || got > 61000 { t.Fatalf("box surface area = %g, want ~60000", got) }
}
```
(Place a real oracle-exported `A1.step` at `model/feature/occtparity/testdata/A1.step`.)

- [ ] **Step 2: Run — fails (importInput undefined)**

Run: `go test ./model/feature/occtparity/ -run TestImportInputBoxArea`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement `importinput.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import (
	"fmt"; "os"
	"oblikovati.org/kernel/exchange"
	stepio "oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// importInput loads OCCT's exact input solid (STEP-exported by the oracle) as our body,
// so the only variable under test downstream is our fillet, not shape construction.
func importInput(stepPath string) (*topo.Body, error) {
	data, err := os.ReadFile(stepPath)
	if err != nil { return nil, fmt.Errorf("read %s: %w", stepPath, err) }
	bodies, _, err := stepio.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil { return nil, fmt.Errorf("import %s: %w", stepPath, err) }
	if len(bodies) != 1 { return nil, fmt.Errorf("import %s: got %d bodies, want 1", stepPath, len(bodies)) }
	return bodies[0], nil
}

func inputArea(b *topo.Body) float64 { return ops.BodyGeometryProperties(b, ops.PropertyQuality()).Area }
```

- [ ] **Step 4: Run — passes**

Run: `go test ./model/feature/occtparity/ -run TestImportInputBoxArea`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/importinput.go model/feature/occtparity/importinput_test.go model/feature/occtparity/testdata/A1.step
git commit -m "test(occt-parity): STEP input import + area pre-check helper"
```

### Task 5: The `checkprops` assertion (tolerance math + drift warning)

**Files:**
- Create: `model/feature/occtparity/checkprops.go`
- Create: `model/feature/occtparity/checkprops_test.go`

**Interfaces:**
- Produces: `func assertArea(t testingT, name string, got, expected, deps float64)` where
  `testingT` is a tiny interface (`Errorf`, `Logf`) so it is unit-testable with a fake.

- [ ] **Step 1: Write the failing test — a fake recorder proves the boundaries**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import "testing"

type fakeT struct{ errs, logs []string }
func (f *fakeT) Errorf(s string, a ...any) { f.errs = append(f.errs, s) }
func (f *fakeT) Logf(s string, a ...any)   { f.logs = append(f.logs, s) }

func TestAssertAreaTolerance(t *testing.T) {
	// within 1% ⇒ no error; >0.1% ⇒ a drift warning
	f := &fakeT{}
	assertArea(f, "c", 100.5, 100.0, 0.01) // 0.5% off: pass, warn
	if len(f.errs) != 0 { t.Fatalf("unexpected error at 0.5%%: %v", f.errs) }
	if len(f.logs) != 1 { t.Fatalf("expected drift warning at 0.5%%, got %d", len(f.logs)) }
	// >1% ⇒ error
	f2 := &fakeT{}
	assertArea(f2, "c", 102.0, 100.0, 0.01) // 2% off
	if len(f2.errs) != 1 { t.Fatalf("expected error at 2%%, got %d", len(f2.errs)) }
	// within 0.1% ⇒ no error, no warning
	f3 := &fakeT{}
	assertArea(f3, "c", 100.05, 100.0, 0.01) // 0.05% off
	if len(f3.errs) != 0 || len(f3.logs) != 0 { t.Fatalf("clean case warned/errored: %v %v", f3.errs, f3.logs) }
}
```

- [ ] **Step 2: Run — fails (assertArea undefined)**

Run: `go test ./model/feature/occtparity/ -run TestAssertAreaTolerance`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement `checkprops.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import stdmath "math"

// testingT is the slice of *testing.T the assertion needs, so it is unit-testable.
type testingT interface {
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
}

const driftWarnRel = 1e-3 // 0.1% — sharper internal signal, non-failing (see spec)

// assertArea mirrors OCCT checkprops: fail when relative area error exceeds deps (OCCT's
// 1% default), warn (non-failing) when it merely exceeds 0.1%. expected==0 is handled by
// the caller (OCCT-TODO -s 0 cases are skipped upstream, never reach here).
func assertArea(t testingT, name string, got, expected, deps float64) {
	rel := stdmath.Abs(got-expected) / stdmath.Abs(expected)
	if rel > deps {
		t.Errorf("%s: area %.6g != OCCT %.6g (rel %.4f%% > %.2f%%)", name, got, expected, rel*100, deps*100)
		return
	}
	if rel > driftWarnRel {
		t.Logf("%s: area drift %.4f%% from OCCT %.6g (within %.1f%% gate)", name, rel*100, expected, deps*100)
	}
}
```

- [ ] **Step 4: Run — passes**

Run: `go test ./model/feature/occtparity/ -run TestAssertAreaTolerance`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/checkprops.go model/feature/occtparity/checkprops_test.go
git commit -m "test(occt-parity): area assertion at OCCT 1% tolerance + 0.1% drift warning"
```

### Task 6: The geometric edge locator

**Files:**
- Create: `model/feature/occtparity/edgepick.go`
- Create: `model/feature/occtparity/edgepick_test.go`

**Interfaces:**
- Consumes: `Locator` (Task 3), `topo.DescribeEdge` (`{Midpoint, Direction}`).
- Produces: `func locateEdge(b *topo.Body, loc Locator, tol float64) (*topo.Edge, error)` —
  the body edge whose midpoint matches `loc.Midpoint` within `tol` (direction breaks ties).

- [ ] **Step 1: Write the failing test** — build a 100³ box (`brep.SolidBlock`), pick the
edge at a known midpoint, and confirm the locator returns exactly it.

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import (
	"testing"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func TestLocateEdgePicksUniqueBoxEdge(t *testing.T) {
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(100, 100, 100), "box")
	if err != nil { t.Fatalf("box: %v", err) }
	// pick any real edge's midpoint as the target, then confirm round-trip.
	want := box.Edges()[3]
	ref := topo.DescribeEdge(want)
	loc := Locator{Midpoint: [3]float64{ref.Midpoint.X, ref.Midpoint.Y, ref.Midpoint.Z}}
	got, err := locateEdge(box, loc, 1e-6)
	if err != nil { t.Fatalf("locate: %v", err) }
	if got.ID() != want.ID() { t.Fatalf("located edge %d, want %d", got.ID(), want.ID()) }
}
```

- [ ] **Step 2: Run — fails (locateEdge undefined)**

Run: `go test ./model/feature/occtparity/ -run TestLocateEdgePicksUniqueBoxEdge`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement `edgepick.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import (
	"fmt"; stdmath "math"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// locateEdge returns the body edge whose midpoint matches the oracle locator within tol.
// OCCT resolved the pick to a concrete edge; we re-find that edge by geometry so we do not
// depend on our own topology's edge ordering. Ties (equal midpoints) break on direction.
func locateEdge(b *topo.Body, loc Locator, tol float64) (*topo.Edge, error) {
	target := math.P3(loc.Midpoint[0], loc.Midpoint[1], loc.Midpoint[2])
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range b.Edges() {
		d := topo.DescribeEdge(e).Midpoint.Sub(target).Length()
		if d < bestD { bestD, best = d, e }
	}
	if best == nil || bestD > tol {
		return nil, fmt.Errorf("locateEdge: no edge within %.3g of midpoint %v (closest %.3g)", tol, loc.Midpoint, bestD)
	}
	return best, nil
}
```
(Direction tie-break: if a later corpus case has two edges sharing a midpoint — none in the
inline set do — extend to compare `Direction`; add only when a real case needs it, YAGNI.)

- [ ] **Step 4: Run — passes**

Run: `go test ./model/feature/occtparity/ -run TestLocateEdgePicksUniqueBoxEdge`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/edgepick.go model/feature/occtparity/edgepick_test.go
git commit -m "test(occt-parity): geometric edge locator (OCCT index -> our edge by midpoint)"
```

### Task 7: Outcome status classification

**Files:**
- Create: `model/feature/occtparity/status.go`
- Create: `model/feature/occtparity/status_test.go`

**Interfaces:**
- Produces: `type Outcome int` (`Pass`, `FailFaulty`, `FailArea`, `Incomplete`,
  `SkipTODO`, `SkipImportDivergence`) and `func classify(r Record, importOK, filletOK, valid bool) Outcome`
  — the single place mapping a run's facts to OCCT's own verdict semantics.

- [ ] **Step 1: Write the failing test**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import "testing"

func TestClassifyMirrorsOCCT(t *testing.T) {
	todo := Record{TODO: "TODO OCC22817 All:TEST INCOMPLETE"}
	if classify(todo, true, false, false) != SkipTODO { t.Fatal("TODO case must skip") }
	ok := Record{}
	if classify(ok, true, true, true) != Pass { t.Fatal("clean run must pass") }
	if classify(ok, true, true, false) != FailFaulty { t.Fatal("invalid solid must fail Faulty") }
	if classify(ok, false, false, false) != SkipImportDivergence { t.Fatal("import failure separates from fillet") }
}
```

- [ ] **Step 2: Run — fails**; **Step 3: implement `status.go`:**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

type Outcome int

const (
	Pass Outcome = iota
	FailFaulty            // result not a valid solid (OCCT \bFaulty\b)
	FailArea             // valid but area outside tolerance (asserted separately)
	Incomplete           // OCCT tolerance-ang IGNORE analogue
	SkipTODO             // OCCT TODO/INCOMPLETE marker present
	SkipImportDivergence // STEP input did not import faithfully — not a fillet defect
)

// classify maps run facts to OCCT's verdict semantics. TODO wins (we never exceed OCCT);
// import divergence is separated from fillet defects; an invalid result is Faulty.
func classify(r Record, importOK, filletOK, valid bool) Outcome {
	switch {
	case r.TODO != "":
		return SkipTODO
	case !importOK:
		return SkipImportDivergence
	case !filletOK || !valid:
		return FailFaulty
	default:
		return Pass
	}
}
```

- [ ] **Step 4: Run — passes.** **Step 5: Commit**

```bash
git add model/feature/occtparity/status.go model/feature/occtparity/status_test.go
git commit -m "test(occt-parity): outcome classification mirroring OCCT parse.rules"
```

### Task 8: The case runner (drive the real fillet feature + assert)

**Files:**
- Create: `model/feature/occtparity/runcase.go`
- Create: `model/feature/occtparity/runcase_test.go`

**Interfaces:**
- Consumes: `importInput`, `inputArea`, `locateEdge`, `assertArea`, `classify`, `Record`.
- Produces: `func RunCase(t *testing.T, r Record, fixtureDir string)` — the one entry each
  grid test calls per case. Skips TODO/import-divergence; otherwise imports input, locates
  picks, drives `AddFillet`/`AddFilletSets`, recomputes, validates, asserts area.

- [ ] **Step 1: Write the failing test** — an end-to-end on a real box case, driving the
feature path, asserting OCCT's area. Use `simple/A1` (`box 100³`, fillet one edge r=10,
`-s 59527.9`), whose STEP + locator you have from Task 1/4.

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import ( "path/filepath"; "testing" )

func TestRunCaseSimpleA1(t *testing.T) {
	r := Record{Grid: "simple", Case: "A1", Verb: "blend", ExpectedArea: 59527.9, Deps: 0.01,
		InputStep: "A1.step",
		Picks: []Pick{{Radius: 10, Locator: Locator{Midpoint: midpointOfA1PickedEdge()}}}}
	RunCase(t, r, "testdata")
}
```
(`midpointOfA1PickedEdge()` returns the midpoint the oracle recorded for `simple/A1`'s
`s_5` — a constant helper in the test; the value comes from `/tmp/oracle-out/A1.json`.)

- [ ] **Step 2: Run — fails (RunCase undefined; and/or our fillet may not yet produce OCCT's area — that is the point)**

Run: `go test ./model/feature/occtparity/ -run TestRunCaseSimpleA1 -v`
Expected: first FAIL to compile (undefined), then after Step 3 either PASS (if our fillet
matches within 1%) or FAIL on area — a **real parity signal**, recorded, not worked around.

- [ ] **Step 3: Implement `runcase.go`**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import (
	"path/filepath"; "testing"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// RunCase drives one OCCT blend case through our real fillet feature and asserts OCCT's
// area. It skips OCCT-TODO and import-divergent cases so the gate reflects fillet parity,
// not import or OCCT-incompleteness. Constant-radius only here; buildevol added in Task 12.
func RunCase(t *testing.T, r Record, fixtureDir string) {
	t.Helper()
	if r.TODO != "" { t.Skipf("OCCT marks incomplete: %s", r.TODO); return }
	body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
	if err != nil { t.Skipf("import-divergence (not a fillet defect): %v", err); return }

	keys := make([][]byte, 0, len(r.Picks))
	sets := make([]feature.FilletEdgeSet, 0, len(r.Picks))
	for _, p := range r.Picks {
		e, err := locateEdge(body, p.Locator, importTol(body))
		if err != nil { t.Fatalf("%s/%s: %v", r.Grid, r.Case, err) }
		radius := p.Radius
		sets = append(sets, feature.FilletEdgeSet{EdgeKeys: [][]byte{e.ReferenceKey()}, Radius: func() float64 { return radius }})
		keys = append(keys, e.ReferenceKey())
	}

	fs := feature.NewPartFeatures(nil)
	feature.NewBaseFeatures(fs).AddBase(body)
	pf := feature.NewDressUpFeatures(fs).AddFilletSets(sets) // per-pick radii (A2 etc. use mixed radii)
	fs.Recompute()

	filletOK := pf.Health().OK()
	res := fs.Result()
	valid := filletOK && len(res) == 1 && res[0] != nil && ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Volume > 0
	switch classify(r, true, filletOK, valid) {
	case FailFaulty:
		t.Fatalf("%s/%s: result not a valid solid: %s", r.Grid, r.Case, pf.Health().Reason)
	case Pass:
		got := ops.BodyGeometryProperties(res[0], ops.PropertyQuality()).Area
		assertArea(t, r.Grid+"/"+r.Case, got, r.ExpectedArea, r.Deps)
	}
}

// importTol scales the locator match tolerance to the body so scaled fixtures (tscale
// SCALE=1000) still resolve; 1e-4 of the body's bounding diagonal.
func importTol(b *topo.Body) float64 { /* return 1e-4 * boundingDiag(b) */ return 1e-4 * boundingDiag(b) }
```
Add a small `boundingDiag(*topo.Body) float64` from `body.Vertices()` extents in the same
file (4–8 lines).

- [ ] **Step 4: Run — record the outcome**

Run: `go test ./model/feature/occtparity/ -run TestRunCaseSimpleA1 -v`
Expected: PASS if our single-edge box fillet is within 1% of OCCT (likely); if it FAILs on
area, that is a genuine finding for the greening backlog — leave the assertion in, do NOT
loosen it.

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/runcase.go model/feature/occtparity/runcase_test.go
git commit -m "test(occt-parity): case runner — real fillet feature path + OCCT area assertion"
```

### Task 9: Generate `corpus.go` from the oracle output (all grids)

**Files:**
- Create: `test-utilities/occt-blend/gen/main.go` (a `go run` generator, package `main`)
- Create: `model/feature/occtparity/corpus.go` (generated; committed)
- Create: `model/feature/occtparity/corpus_test.go` (invariants on the generated table)

**Interfaces:**
- Consumes: the oracle binary (Task 1/3) over all 477 `.tcl` files + fixtures (Task 2).
- Produces: `func Corpus() []Record` and `func CorpusFixtureDir() string`; committed STEP
  fixtures under `model/feature/occtparity/fixtures/<grid>/<case>.step`.

- [ ] **Step 1: Write the failing invariant test**

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import "testing"

func TestCorpusIsComplete(t *testing.T) {
	c := Corpus()
	if len(c) != 477 { t.Fatalf("corpus has %d cases, want 477", len(c)) }
	for _, r := range c {
		if r.Grid == "" || r.Case == "" { t.Fatalf("case missing grid/name: %+v", r) }
		if r.TODO == "" && r.ExpectedArea <= 0 { t.Fatalf("%s/%s non-TODO but expectedArea<=0", r.Grid, r.Case) }
		if r.TODO == "" && len(r.Picks) == 0 { t.Fatalf("%s/%s non-TODO but no picks", r.Grid, r.Case) }
	}
}
```

- [ ] **Step 2: Run — fails (Corpus undefined / count 0)**

Run: `go test ./model/feature/occtparity/ -run TestCorpusIsComplete`
Expected: FAIL.

- [ ] **Step 3: Write the generator** — iterate `../OCCT/tests/blend/<grid>/<case>`, run the
oracle into a temp dir, copy each `<case>.step` to `fixtures/<grid>/<case>.step`, parse each
`<case>.json` into a `Record`, and emit `corpus.go` as a Go source literal (`gofmt`ed).
Unparsed/oracle-errored cases are emitted as `TODO:"unparsed: <err>"` records — loud, never
dropped (the invariant test count stays 477).

```go
// SPDX-License-Identifier: GPL-2.0-only
package main
// go run ./test-utilities/occt-blend/gen -occt ../OCCT/tests/blend -oracle <bin> -out model/feature/occtparity
// Walks every grid/case, runs the oracle, writes fixtures/<grid>/<case>.step and corpus.go.
func main() { /* … os/exec the oracle, template corpus.go … */ }
```

- [ ] **Step 4: Run the generator, then the invariant test**

Run: `go run ./test-utilities/occt-blend/gen -occt ../OCCT/tests/blend -oracle test-utilities/occt-blend/oracle/build/occt_blend_oracle -out model/feature/occtparity && go test ./model/feature/occtparity/ -run TestCorpusIsComplete`
Expected: PASS — 477 records, fixtures written.

- [ ] **Step 5: Commit** (generator + generated corpus + fixtures)

```bash
git add test-utilities/occt-blend/gen model/feature/occtparity/corpus.go model/feature/occtparity/fixtures
git commit -m "test(occt-parity): generate 477-case corpus.go + STEP fixtures from the OCCT oracle"
```

---

## Phase 2 — Wire the full corpus + scoreboard

### Task 10: Per-grid table tests over the generated corpus

**Files:**
- Create: `model/feature/occt_blend_simple_test.go`
- Create: `model/feature/occt_blend_buildevol_test.go`
- Create: `model/feature/occt_blend_bfuse_test.go`
- Create: `model/feature/occt_blend_tolblend_test.go`
- Create: `model/feature/occt_blend_complex_test.go`
- Create: `model/feature/occt_blend_encreg_test.go`

**Interfaces:**
- Consumes: `occtparity.Corpus()`, `occtparity.RunCase`, `occtparity.CorpusFixtureDir()`.

- [ ] **Step 1: Write the `simple`-grid table test** (the pattern; one file per grid)

```go
// SPDX-License-Identifier: GPL-2.0-only
package feature

import (
	"testing"
	"oblikovati.org/model/feature/occtparity"
)

func TestOCCTBlendSimple(t *testing.T) {
	for _, r := range occtparity.Corpus() {
		if r.Grid != "simple" { continue }
		r := r
		t.Run(r.Case, func(t *testing.T) { occtparity.RunCase(t, r, occtparity.CorpusFixtureDir()) })
	}
}
```
Package `feature` (not `occtparity`) so `RunCase` exercises the real feature layer from
outside its own package. This requires `RunCase`, `Corpus`, `Record`, `CorpusFixtureDir`,
`FilletEdgeSet` usage to be exported (they are).

- [ ] **Step 2: Run — expect many reds (constant-radius corners, curved neighbours) — that is the scoreboard**

Run: `go test ./model/feature/ -run TestOCCTBlendSimple -v 2>&1 | tail -40`
Expected: a mix of PASS/FAIL/SKIP; **do not fix reds here** — they are the greening
backlog. Confirm the harness itself runs every case without panicking.

- [ ] **Step 3: Add the other five grid files** (identical shape, `r.Grid` switched to
`buildevol`/`bfuseblend`/`tolblend_simple`+`tolblend_buildvol`/`complex`/`encoderegularity`).
`buildevol` cases will SKIP or FAIL until Task 12; note that in the file's doc comment.

- [ ] **Step 4: Run the whole corpus once**

Run: `go test ./model/feature/ -run TestOCCTBlend 2>&1 | tail -20`
Expected: runs to completion (reds allowed); no panics, no harness crashes.

- [ ] **Step 5: Commit**

```bash
git add model/feature/occt_blend_*_test.go
git commit -m "test(occt-parity): per-grid table tests over the OCCT blend corpus (scoreboard live)"
```

### Task 11: Scoreboard reporter

**Files:**
- Create: `model/feature/occtparity/scoreboard.go`
- Create: `model/feature/occt_blend_scoreboard_test.go`

**Interfaces:**
- Produces: `func Scoreboard(records []Record, run func(Record) Outcome) map[Outcome]int`
  and a `TestOCCTBlendScoreboard` that prints per-grid Pass/Fail/Skip counts (via `t.Log`)
  and is **non-gating** (it summarizes; the per-grid tests are the gate).

- [ ] **Step 1: Write the failing test** — a fake run function yields a known tally.

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import "testing"

func TestScoreboardTally(t *testing.T) {
	recs := []Record{{Grid: "simple"}, {Grid: "simple"}, {Grid: "simple", TODO: "x"}}
	got := Scoreboard(recs, func(r Record) Outcome {
		if r.TODO != "" { return SkipTODO }; return Pass
	})
	if got[Pass] != 2 || got[SkipTODO] != 1 { t.Fatalf("tally: %+v", got) }
}
```

- [ ] **Step 2: Run — fails; Step 3: implement `scoreboard.go`** (a plain tally loop, 6–10
lines) and the reporter test that runs each corpus record through a non-asserting variant of
the runner and logs the per-grid table. **Step 4: Run** —

Run: `go test ./model/feature/ -run TestOCCTBlendScoreboard -v 2>&1 | tail -20`
Expected: prints a per-grid Pass/Fail/Skip table; test itself passes (non-gating).

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/scoreboard.go model/feature/occtparity/scoreboard_test.go model/feature/occt_blend_scoreboard_test.go
git commit -m "test(occt-parity): non-gating scoreboard reporter (per-grid Pass/Fail/Skip)"
```

### Task 12: Variable-radius (buildevol) support in the runner

**Files:**
- Modify: `model/feature/occtparity/runcase.go`
- Create: `model/feature/occtparity/runcase_evol_test.go`

**Interfaces:**
- Consumes: `Pick.Law [][2]float64`, `ops.EdgeFilletRadii`, `ops.FilletRadiusPoint`,
  `ops.FilletEdgesVarying` (kernel) OR a feature-layer variable-radius Add if one exists.
- Produces: `RunCase` handling `r.Verb == "buildevol"` by mapping each pick's `Law` to a
  varying-radius fillet.

- [ ] **Step 1: Investigate whether the feature layer exposes variable radius** — grep
`model/feature` for a `AddFilletVarying`/`Mids`/`EdgeAnchors` public entry. If present, use
it (real feature path). If absent, this is an **API gap**: file it against M44/#1887's
neighbourhood and, for now, drive `ops.FilletEdgesVarying` directly on the imported body
inside the runner (documented as a temporary kernel-level path until the feature entry
exists). Record the decision in the file's doc comment.

- [ ] **Step 2: Write the failing test** on `buildevol/A5` (`box 100 100 10`, `updatevol
s_5 0 2 1 4 2 2`, `-s 23985.2`).

```go
// SPDX-License-Identifier: GPL-2.0-only
package occtparity

import ( "path/filepath"; "testing" )

func TestRunCaseBuildevolA5(t *testing.T) {
	r := Record{Grid: "buildevol", Case: "A5", Verb: "buildevol", ExpectedArea: 23985.2, Deps: 0.01,
		InputStep: "buildevol_A5.step",
		Picks: []Pick{{Locator: Locator{Midpoint: midpointOfA5PickedEdge()}, Law: [][2]float64{{0, 2}, {1, 4}, {2, 2}}}}}
	RunCase(t, r, "testdata")
}
```

- [ ] **Step 3: Extend `runcase.go`** to branch on `r.Verb`: build
`ops.EdgeFilletRadii{Key, R0: law[0][1], R1: law[last][1], Mids: mapLaw(law)}` and drive the
chosen variable-radius path; keep constant-radius unchanged.

- [ ] **Step 4: Run — record outcome** (PASS within 1% or a real parity FAIL for the backlog)

Run: `go test ./model/feature/occtparity/ -run TestRunCaseBuildevolA5 -v`
Expected: compiles + runs; assertion reflects true parity.

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/runcase.go model/feature/occtparity/runcase_evol_test.go model/feature/occtparity/testdata/buildevol_A5.step
git commit -m "test(occt-parity): variable-radius (buildevol) cases through the runner"
```

### Task 13: Documentation + final corpus regeneration

**Files:**
- Create: `model/feature/occtparity/README.md`
- Modify: `docs/superpowers/specs/2026-07-11-occt-blend-parity-corpus-design.md` (note the oracle-STEP refinement)

**Interfaces:** none (docs).

- [ ] **Step 1: Write `README.md`** — how the oracle→generator→corpus pipeline works, how to
regenerate (`go run ./test-utilities/occt-blend/gen …`), how to read the scoreboard, and the
rule that **reds are the greening backlog, never loosened**.

- [ ] **Step 2: Regenerate the corpus from scratch and run the full suite once**

Run: `go run ./test-utilities/occt-blend/gen -occt ../OCCT/tests/blend -oracle test-utilities/occt-blend/oracle/build/occt_blend_oracle -out model/feature/occtparity && go test ./model/feature/ -run TestOCCTBlend 2>&1 | tail -30`
Expected: 477 cases exercised; scoreboard stable; harness green (assertions may be red —
that is the tracked backlog, not a harness failure).

- [ ] **Step 3: Commit**

```bash
git add model/feature/occtparity/README.md docs/superpowers/specs/2026-07-11-occt-blend-parity-corpus-design.md
git commit -m "docs(occt-parity): corpus pipeline README + spec refinement note"
```

---

## Out of scope (tracked separately)

- **Greening the red cases** — the engine work (constant-radius corners incl.
  `IntersectionAtEnd`/n-way, curved neighbours, variable-radius laws, setback pending M44
  #1887) that turns the scoreboard green. Each is its own spec/plan grounded in
  `geometry-math-advisor` + ADR-0050. The single PR for this whole milestone lands only when
  the scoreboard is green (excluding OCCT-TODO).
- **chamfer (`tests/chamfer/`) and `fillet2d` grids** — same harness, follow-up.

## Self-review notes

- **Spec coverage:** harness (checkprops/edgepick/status/runcase) ✓; generator ✓; full
  477 port ✓; scoreboard ✓; area-only + 1% + drift-warn ✓; TODO mirroring ✓; feature-path
  drive ✓; data-file cases via oracle-STEP ✓. The spec's `draw.go`/`profile.go` DSL is
  deliberately dropped (Task header explains) — superseded by oracle-STEP, which covers the
  restore cases the DSL could not.
- **Deviation flagged:** oracle-STEP architecture refines the approved spec; noted in the
  plan header and Task 13 updates the spec.
- **Risk — Phase 0 is a spike gate:** if the oracle cannot be built/linked, escalate before
  Phase 1 (the whole plan depends on it). The fallback (enable OCCT's own DRAWEXE in
  `occt-build`) is named in Task 1.
- **Type consistency:** `Record`/`Pick`/`Locator`/`Outcome`/`RunCase`/`Corpus`/`assertArea`/
  `locateEdge`/`classify`/`importInput` names are used identically across Tasks 3–13.
