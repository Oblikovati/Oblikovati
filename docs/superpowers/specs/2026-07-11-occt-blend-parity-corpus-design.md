# OCCT Blend Parity Test Corpus — Design

**Date:** 2026-07-11 · **Author:** brainstormed with vmiguel · **Milestone:** successor to
M41 (OCCT fillet/chamfer/draft parity, ADR-0050) · **Drives:** completion of the ADR-0050
blend engine, gated by OCCT's own test corpus.

## Goal

Port OpenCASCADE's `tests/blend/*` test corpus (~477 cases) into the Oblikovati Go test
suite as **faithful, hard-asserting** regression tests, and drive the fillet/blend engine
until **every ported case passes at OCCT's own tolerance**. The ported corpus is
simultaneously (a) the parity spec — OCCT's numbers are the source of truth — and (b) the
regression gate for all downstream corner/variable-radius engine work.

## Why this, why now

Two verified ground-truth findings reframed the original "add corner-blend support" ask:

1. **The ADR-0050 blend engine is partially wired and has no corner module.**
   `kernel/blend/` exists (marcher, known-part, spine, section functionals, status) but
   `kernel/blend/corner.go` **does not exist in `develop`** despite issue #1808 (Phase 6)
   being closed. The engine's `Builder` is not called by the feature layer at all — only
   `blend.Marcher`, via the tangent-stripe path (`kernel/ops/fillet_stripe.go`). Real
   corner handling lives in the *old* `kernel/ops` catalog and covers only **2-edge miter
   + 3-edge sphere** corners: no n-way, and no `IntersectionAtEnd` (terminating a stripe
   against a pre-existing round). rampam's #1797 case is short-circuited by
   `curvedEndpointError` before any surface is built.

2. **Our fillet coverage is a catalog of hand-picked scenarios, not a parity suite.**
   "Tests can have bad premises and pass" (CLAUDE.md). We cannot know which fillet
   combinations actually work until we assert against an external oracle. OCCT is that
   oracle (CLAUDE.md conflict-resolution + oracle rules). Porting its corpus converts
   speculation about gaps into an evidence-based, always-current scoreboard.

Therefore the corpus port is the **foundation** of the parity milestone: it defines "done"
for every engine phase that follows, and no engine phase is complete until its cases go
green.

## Scope

**In scope (this design):** the test harness, the case-table generator, the full port of
the self-contained `blend/*` grids, faithful assertion + result semantics, and the
scoreboard tooling. **The engine work to turn red cases green is enumerated here as the
downstream roadmap but each phase gets its own spec** once the scoreboard reports real
pass/fail data.

**The corpus, by capability family** (what the DRAW command vocabulary revealed):

| Grid | Cases | Command family | Capability required |
|---|---|---|---|
| `simple` | 226 | `blend` (constant radius) | constant-radius fillet incl. corners, n-way vertices |
| `buildevol` | 98 | `mkevol`/`updatevol`/`buildevol` | **variable-radius (evolutive) law blends** (ADR-0050 P5) |
| `tolblend_simple` | 56 | `blend` under `tscale` | tolerance / model-scale robustness |
| `bfuseblend` | 16 | `bfuseblend` | boolean-fuse then blend the resulting edges |
| `tolblend_buildvol` | 10 | evolutive under `tscale` | variable radius + tolerance |
| `encoderegularity` | 7 | `encoderegularity` + `blend` | edge-tangency regularity encoding |
| `complex` | 64 | `restore [locate_data_file *.rle]` | **external data-file fixtures**; several are OCCT-TODO |

Self-contained (built from primitives/profiles): `simple + buildevol + tolblend_* +
bfuseblend + encoderegularity` ≈ **413 cases** — portable immediately. The `complex` 64
depend on OCCT `.rle`/`.brep` fixtures not currently checked out (`CFI_indusfjm.rle` absent
locally) and are ported once their data is fetched; OCCT-TODO cases are mirrored (below).

**Non-goals:** chamfer (`tests/chamfer/`) and 2D fillet (`tests/fillet2d/`) grids are
follow-ups on the same harness, not this design. Draft is ADR-0050 P7, separate.

## Faithful-port semantics (the load-bearing decisions)

These are locked; they define what "faithful" means so the port is not re-litigated
case-by-case.

- **Assertion basis = the body's mass properties (the `sprops`/`vprops` analogue), not
  tessellation.** OCCT's `checkprops -s`/`-v` read `sprops`/`vprops` (surface area /
  volume mass properties). We reuse **our existing body physical-properties path**
  (the BodyInfo surface-area + volume mass-props from #1078) — the same numbers the app
  reports — as the faithful analogue, integrated over each **trimmed** face domain (not
  whole-surface `geom.SurfaceArea`, which ignores trimming). This is deliberately the
  production mass-props code, so a divergence is a real geometry/trim defect, not a
  harness artifact. Tessellation area would fail on mesh density, not geometry.
- **Tolerance = OCCT's own.** OCCT's `checkprops`
  (`resources/DrawResources/CheckCommands.tcl:86`) compares
  `abs((expected-actual)/expected) > depsilon`, `depsilon = 1e-2` (1% relative) by
  default, overridable per-test with `-deps`. We use **the test's `-deps` if present, else
  1% relative** — exactly what OCCT passes. Additionally, when our value is within 1% but
  **drifts >0.1% from OCCT's reference**, emit a non-failing `t.Log` warning — a sharper
  internal signal of geometric drift without making the gate stricter than OCCT.
- **Validity mirrors `blend/parse.rules`.** A result that is not a valid closed solid
  (our `Validate` + manifold/orientation check — the analogue of OCCT's `Faulty`)
  **fails**. A blend that raises the tolerance-angle exception OCCT marks `IGNORE` is
  recorded **incomplete**, not failed.
- **Mirror OCCT's own TODO/INCOMPLETE markers.** A case whose `.tcl` carries `puts "TODO
  … TEST INCOMPLETE"` is ported as **expected-incomplete** (Go `t.Skip` with the OCCT TODO
  string as reason). We are never stricter than OCCT on cases OCCT itself does not pass.
- **Edge selection by geometric locator, not explode index.** OCCT names edges by
  `explode e` order (`s_5`). Reproducing OCCT's internal ordering is brittle; instead each
  OCCT index is resolved to **a geometric locator** (the edge whose midpoint/direction
  matches OCCT's edge for that primitive). Faithful to *which* edge, robust to our
  topology's ordering. The generator computes each locator once from the primitive
  definition.
- **Hard gate, single eventual PR.** Every case is a real assertion from day one (no
  silent skips except OCCT-TODO). The branch is long-lived with **granular commits per
  capability** (CLAUDE.md), but **no PR is opened until the entire ported corpus is
  green**. The scoreboard tracks progress on the branch.

## Architecture

Four units with clean boundaries. All test-scope (`_test.go` + a test-only helper
package); zero production dependency, so the harness cannot leak into shipped code.

Placement is at the **feature layer**, not `kernel/ops`: the harness must drive the real
`FilletFeature` recompute path (per `validate-fillet-through-feature-not-ops` — a fix is
not proven by raw `ops` calls), so it imports `model/feature`. Putting it under
`kernel/ops` would invert layering (kernel depending on model). It lives beside the
feature it exercises:

```
model/feature/occtparity/              # test-only helper pkg; imports model/feature + kernel/geom,topo
  draw.go        # DRAW-verb DSL: box/pcylinder/pcone/psphere/ptorus/wedge/prism/revol + transforms
  profile.go     # profile builders: polyline/wire/mkedge/circle/ellipse/bezier/bspline → face/wire
  blendcmd.go    # blend(), mkevol/updatevol/buildevol(), bfuseblend() → AddFillet + Recompute (real feature path)
  edgepick.go    # geometric edge locator: OCCT index -> our topo.Edge by matching midpoint/dir
  checkprops.go  # massProps() [reuse BodyInfo physical props], assertProps(expected, deps, ref-drift warn)
  status.go      # Faulty/Incomplete classification mirroring blend/parse.rules
  corpus.go      # generated case table: []Case{Name, Grid, Build func, Blend spec, ExpArea, ExpVol, Deps, TODO}
  corpus_gen.go  # (generator, run via go:generate) parses OCCT .tcl -> corpus.go
model/feature/occt_blend_simple_test.go   # table-driven: iterate corpus by grid, assert
model/feature/occt_blend_buildevol_test.go
model/feature/occt_blend_bfuse_test.go
... one _test.go per grid ...
test-utilities/occt-blend/             # copied OCCT .tcl sources + provenance (SOURCES.md) the generator reads
```

### Data flow

1. `go:generate` runs `corpus_gen.go`: reads `test-utilities/occt-blend/<grid>/<case>`
   (verbatim OCCT `.tcl`), parses the shape-construction + `blend`/`buildevol` +
   `checkprops` lines, computes the geometric edge locators, and emits `corpus.go` — a
   pure Go table. Re-runnable when OCCT updates. Parsing failures are loud (a case that
   the generator cannot parse is emitted as a `TODO: unparsed` entry, never silently
   dropped).
2. Each grid's `_test.go` iterates its corpus slice: builds the shape via `draw.go`,
   resolves edges via `edgepick.go`, runs the blend via `blendcmd.go` (which calls the
   real feature/kernel fillet path — the same entry the UI uses, per
   `validate-fillet-through-feature-not-ops`), then `checkprops.go` asserts area/volume.
3. `status.go` classifies the outcome (pass / faulty-fail / incomplete / OCCT-TODO-skip).
   A `-run TestOCCTBlend -v` run prints the scoreboard; a `go test` failure lists every
   red case with its OCCT reference vs our value.

### Boundaries

- `occtparity` imports `model/feature` (to drive the real fillet path) plus
  `kernel/geom`, `kernel/topo`. It is imported **only** by `_test.go` files and never
  ships. Layering flows model→kernel as usual; the harness sits at the model/feature
  layer, so there is no inversion.
- The **generator** owns OCCT-syntax knowledge; the **runtime table** (`corpus.go`) is
  plain data. Changing how we parse OCCT never touches the assertion logic and vice versa.
- `edgepick.go` is the single place that knows the OCCT-index → geometry mapping; if our
  topology ordering changes, only this file cares.

## Downstream greening roadmap (each its own spec later)

The scoreboard from the first run defines the real gaps. Anticipated phases, grounded to
OCCT `ChFi3d` and the corpus families:

- **G1 — constant-radius corners (`simple` red cases).** `IntersectionAtEnd`
  (`PerformOneCorner`: stripe end trimmed into a pre-existing round — retires
  `curvedEndpointError`, fixes #1797) and n-way vertex reconstruction
  (`PerformThreeCorner`/`MoreThreeCorner`) in `kernel/blend/corner.go`, wired via
  `Builder`. Routes numerics through `geometry-math-advisor`.
- **G2 — evolutive/variable-radius (`buildevol`).** `updatevol` law → `EvolRadiusSection`
  through the marcher (ADR-0050 P5 completion).
- **G3 — tolerance/scale (`tolblend_*`).** Model-relative tolerance robustness under
  `tscale` (ADR-0042).
- **G4 — bfuseblend + encoderegularity.** Boolean-then-blend edge provenance;
  regularity encoding.
- **G5 — complex fixtures.** Fetch OCCT data files; port + mirror TODOs.

The single PR lands when G1–G5 (excluding OCCT-TODO) are green.

## Testing

The corpus *is* the test. Beyond it: unit tests for the harness itself —
`edgepick.go` (locator resolves the intended edge on each primitive),
`checkprops.go` (tolerance math matches OCCT's formula exactly, including the >0.1% drift
warning boundary), `status.go` (Faulty vs Incomplete vs TODO classification), and
`corpus_gen.go` (parses a representative case of each command family correctly).

## Risks

- **Edge-locator ambiguity** on symmetric primitives (a cube's 12 edges fall into 3
  geometric classes). Mitigation: the locator keys on midpoint position, not just
  direction, which is unique per edge on all corpus primitives; the generator asserts each
  index resolves to exactly one edge and fails loudly otherwise.
- **`restore` fixtures unavailable.** The `complex` grid needs external data. Mitigation:
  scoped to G5; the other 413 cases are self-contained.
- **Valid-but-different geometry.** Our corner surface may differ from OCCT's within 1% —
  acceptable by design (1% gate = OCCT's gate). The >0.1% drift warning surfaces these for
  review without failing.
- **Long red period.** The gate stays red until G1–G5 land — by explicit decision (no PR
  until green). The scoreboard makes progress visible on the branch.

## Provenance

OCCT `tests/blend/*` (self-contained cases) copied verbatim into
`test-utilities/occt-blend/` with `SOURCES.md` recording the OCCT commit. Tolerance and
validity semantics grounded in `resources/DrawResources/CheckCommands.tcl` and each grid's
`parse.rules`. Aligns with ADR-0050 (engine) and the CLAUDE.md OCCT-oracle rule.
