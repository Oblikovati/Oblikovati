<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OCCT blend-parity oracle

Runs one OpenCASCADE `tests/blend/<grid>/<case>` script unmodified and emits, per case:

- `<case>.step` — OCCT's **exact input solid** (the shape `blend` filleted), STEP-exported
  so our kernel imports identical geometry and the only variable under test is *our fillet*.
- `<case>.json` — the pick list (radius + a geometric edge locator), OCCT's reference area,
  the tolerance (`deps`), and any TODO marker. Schema frozen in Task 3 (`schema.json`).

## Usage

```bash
test-utilities/occt-blend/oracle/occt_blend_oracle ../OCCT/tests/blend/simple/A1 /tmp/oracle-out
# writes /tmp/oracle-out/A1.step and /tmp/oracle-out/A1.json
```

## Design decision (Task 1 spike): DRAWEXE + Tcl helper, not a bespoke C++ binary

The plan permitted either a hand-written C++ program embedding a Tcl interpreter **or**
driving OCCT's own DRAWEXE via a sourced Tcl helper, "decided by spike." The spike chose
the **DRAWEXE + Tcl** path decisively:

- DRAWEXE is already built (`occt-build/lin64/gcc/bin/DRAWEXE`, see `drawenv.sh`); the C++
  path needs a fresh CMake target linking ~15 OCCT toolkits.
- Every primitive the oracle needs is already a DRAW command in the built toolkits:
  - `stepwrite a <shape> <file>` (TKXSDRAWSTEP) exports the input solid;
  - an edge's **true mid-parameter point + unit tangent** comes from `mkcurve` + `bounds`
    + `cvalue` (TKTopTest) — this is OCCT's own curve evaluation, a geometry-only locator
    independent of either kernel's edge ordering.
- We intercept OCCT's own `blend` / `checkprops` commands (record args, then delegate), so
  OCCT resolves the picked edges exactly as the test does. No re-implementation of OCCT's
  shape DSL, no restored-fixture reconstruction — `restore [locate_data_file …]` cases work
  the moment the fixtures are vendored (Task 2), because DRAWEXE runs the case verbatim.

So there is **no `occt_blend_oracle.cxx` / `CMakeLists.txt`**; the oracle is `oracle.tcl`
(the driver, sourced by DRAWEXE) plus `occt_blend_oracle` (a thin bash wrapper that sets the
DRAWEXE environment from `drawenv.sh` and passes the case via `ORACLE_*` env vars, since
DRAWEXE batch mode gives a sourced script no argv).

## Hard-won DRAW gotchas (encoded in `oracle.tcl`)

1. **DRAWEXE `-b` executes stdin line-by-line** — multi-line `proc`/`expr` break. All logic
   lives in `oracle.tcl`, fed to DRAWEXE as a single `source` (which buffers the whole file).
2. **DRAW shapes are Tcl *global* variables.** A `proc` that references a shape (the input
   edge, or a temp curve) must declare it `global`, or the shape is invisible and the command
   fails with an empty error message. See `edgeloc`.
3. **`dval` resolves DRAW variable names through the current Tcl scope.** Grid variables
   (`SCALE`, `SCALE1`, …) set by `begin` are globals, so a radius expression like `0.5*SCALE1`
   evaluated inside the `blend` override reads `SCALE1` as 0 unless evaluated at global scope
   (`uplevel #0`). This bug produced early `radius:0` records; the fix is in the `blend` proc.

## Scope

Task 1 handles the **inline-primitive constant-`blend`** form (`box`/`pcylinder`/… +
`explode` + `blend result s r e …` + `checkprops -s`), validated on `simple/{A1,A8,F9,K3}`
(single/multi-edge, `SCALE`-expression radii, curved-boolean input) and the TODO case
`simple/U7`. `mkevol`/`updatevol`/`buildevol` (variable radius) and `bfuseblend` are added in
Task 3; external `locate_data_file` fixtures in Task 2.
