<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Curved-arm single-edge runout — implementation plan (R1–R4)

> **For agentic workers:** execute via superpowers:subagent-driven-development — fresh implementer
> per task (opus), adversarial reviewer per task (fable), mesh-hash byte-identity gate. The evidence,
> DRAWEXE certificates, and the reuse map live in `.superpowers/sdd/curved-runout-forensic.md`
> (the Phase-1 forensic) and `.superpowers/sdd/far-runout-engine-architecture.md` (the engine seam).
> This plan is the task decomposition and the binding global constraints; each task brief is written
> from the forensic.

**Goal:** green OCCT `blend/simple/{B6,C1,C5,C9,D4,D8,E3,M7}` (corpus 64→72) — the eight **single-edge
curved-arm runouts** (one convex `Plane∧{Cylinder|Cone|Sphere}` edge, two trihedral plane-capped ends,
no other pick) — with the exact rolling-ball fillet, by giving the already-correct far-runout
termination engine a single-arm caller.

**Architecture:** the 8 are NOT trihedral corners; the "needs 3 arms" floor
(`fillet_curved_weld.go:60`) is a dispatch mis-classification. The per-end termination geometry they
need is already implemented and correct (`armFarRunout`→`perpendicularRunout`/`obliqueRunout`→
`intersectArmCapping`), but is gated behind the 3-arm trihedral weld and never reached for a single
arm. This slice adds (a) a single-arm-runout **dispatch** ahead of the 3-arm floor, and (b) a
corner-free **both-ends assembly** — an arm face bounded by `[hostA, farTrim@cap₂, hostB(rev),
farTrim@cap₁]` (no setback great-arc, no corner sphere) welded to its two receded hosts. It reuses
`armFarRunout` per end by synthesizing a minimal `cornerWeld{center: otherEndPoint, radius: r}` (the
ONLY two fields `armFarRunout` reads). No new numerics.

**Tech Stack:** Go, `kernel/ops` (GPL-2.0-only), `kernel/geom`, `math`; DRAWEXE-8.0.0 oracle; the
`model/feature/occtparity` corpus gate.

## Global Constraints (bind every task — verbatim from the campaign)

- **NO PR** — corpus far from whole-green (72/195 after this slice); commit-on-branch only.
- **BYTE-IDENTITY of every prior green is a HARD gate, at the MESH-BIT level, NOT verdict-set.** Each
  task compares, base worktree vs HEAD, the **order-independent commutative triangle-bit fingerprint**
  (mod-2⁶⁴ sum of per-triangle FNV-64a over the sorted 24-byte vertex encoding) + per-body **VOLUME**
  of every prior-green body, using the committed shared helper
  (`model/feature/occtparity/fingerprint_test.go`). ONLY the 8 newly-greened cases may change. B3 vol
  190756.470897507 / N7 vol 963883.383205631 unchanged. The dispatch MUST route ONLY the single-arm
  case (`len(fils)==1`, curved arm, both ends trihedral single-plane caps) — the 3-arm trihedral weld
  path stays bit-identical (it is where every prior curved green lives).
- **EXACT arm surface — no BSpline approximation.** All 8 arms are exact (`cylinderArmSurface` /
  `torusArmSurface` from cone-cap and sphere hosts). Where OCCT emits a rational BSpline (D4/D8/E3),
  Oblikovati ships the exact torus band; the area matches OCCT's BSpline within `deps` (0.01) — do NOT
  down-convert to a BSpline to mimic OCCT.
- **Do-no-harm / honest-reject:** any far vertex that is not a clean trihedral single-plane cap (≥2
  non-host transverse faces, a non-plane cap, or a SECOND picked edge at the vertex — fillet-fillet)
  keeps flooring via the existing `cappingFaceAtFarVertex` admission gate; the single-arm dispatch
  must not swallow those. Decline messages carry the offending value + expected shape.
- **Tessellation correctness is the highest priority.** Each newly-greened case: watertight
  (`Valid`+`HolesContained`+`IsSolid`, every edge exactly 2-incident), volume-positive, and every face
  meshes to its true area with NO FOLDS (`FoldEdgeCount == 0`, the `d5_e4_tessellation_test.go`/D9
  pattern; add per-case assertions).
- Model-relative tolerances only (ADR-0042: `ResolutionForBody`/`res.Weld()·scale`); no bare `1e-6`.
  Funcs 4–20 lines; files <500; explicit types; ≤2 indent; SPDX GPL-2.0-only; `math.P3`.

---

## File Structure

- **Create** `kernel/ops/fillet_curved_single_runout.go` (<500) — the single-arm dispatch classifier
  (`isSingleArmRunout`) + the corner-free assembly (`singleArmRunoutBody`, the both-ends rail loop
  `closeSingleArmRunoutRails`, the per-end `cornerWeld` synthesis) + host retrim reusing
  `curvedHostFaces`. One responsibility: the single-arm-runout weld.
- **Create** `kernel/ops/fillet_curved_single_runout_test.go` — unit tests (dispatch classification;
  the both-ends loop closes; per-end perp/oblique; reject boundary).
- **Modify** `kernel/ops/fillet_curved_weld.go:50-62` — in `assembleCurvedArmBody`, route to
  `singleArmRunoutBody` when `isSingleArmRunout` holds, BEFORE the 3-arm floor. ~5 lines, guarded.
- **Create/extend** `model/feature/occtparity/curved_runout_test.go` — per-case watertight + fold +
  Girard/area assertions for the 8; the byte-identity pin over all prior greens.

---

## Tasks

- **R1 — Dispatch + perpendicular both-ends assembly (TRACER: green B6 + C9).**
  Add `isSingleArmRunout(fils []edgeFillet) bool` (exactly one pick; `curvedArmsOf` returns 1; both end
  vertices pass `cappingFaceAtFarVertex` as trihedral single-plane caps). Route from
  `assembleCurvedArmBody` to a new `singleArmRunoutBody` before the "needs 3 arms" floor. Build the
  arm's two host contact rails (`armHostContactRail` — `curvedHostArc` for a torus arm, the
  `cylinderRulingOuterOnHost` ruling for a cylinder arm) and terminate BOTH ends through `armFarRunout`
  by synthesizing `cornerWeld{center: <other-end vertex point>, radius: r}` per end (the only fields it
  reads). Close `[hostA, farTrim@cap₂, hostB(rev), farTrim@cap₁]` (no setback, no sphere), emit the arm
  face + retrimmed hosts via `curvedHostFaces`, assemble + certify (`Validate`+`HolesContained`+
  `IsSolid`), floor on any decline (`curvedArmUnweldedError`). **Tracer scope = the two arm kinds:** B6
  (cylinder arm, perp/perp) and C9 (cone→torus arm, perp/perp) — proves the loop + assembly end-to-end
  for both. **Gate:** B6, C9 GREEN (corpus 66), watertight + volume-positive + every face fold-free,
  areas within `deps` of OCCT (B6 fillet 1823.48, C9 1298.13); the other 6 still floor honestly (their
  turn is R2/R3); byte-identity of all 64 prior greens; lint 0.

- **R2 — Generalize the perpendicular path (green M7, C1, C5, D8).**
  Bring the remaining four all-perpendicular cases through the R1 construction: M7 (cylinder arm whose
  host B flush-cuts the cylinder through its axis; one unrelated footprint hole survives — keep it),
  C1/C5 (cone→torus, top/base cap), D8 (sphere→torus). Fix any host-retrim / rail-landing edge case the
  four surface — do NOT special-case per fixture; the dispatch and loop stay uniform. **Gate:** M7, C1,
  C5, D8 GREEN (corpus 70), watertight + fold-free, areas within `deps` (M7 1150.26, C1 2469.82, C5
  2252.38, D8 1915.0 — our exact torus vs OCCT's BSpline); byte-identity of the 64 prior greens + B6/C9;
  lint 0.

- **R3 — Oblique end (green D4, E3).**
  Wire the oblique branch (already built: `obliqueRunout`→`intersectArmCapping` torus∩plane spiric) into
  the both-ends loop so a per-end oblique cap terminates the arm and re-terminates the two host rails on
  the analytic feet (`armRunoutFeet`/`reterminateRail`). D4 is oblique/oblique (|t·n|=0.5 both ends), E3
  is mixed perp(pole)/oblique(cap) — proving the per-END (not per-arm) regime dispatch. Guard the loop
  closure so a mixed-regime arm still closes on byte-identical feet. **Gate:** D4, E3 GREEN (corpus 72),
  watertight + fold-free, areas within `deps` (D4 fillet 5263.63, E3 6402.47); byte-identity of all
  prior greens + B6/C9/M7/C1/C5/D8; lint 0.

- **R4 — Reject-boundary regression + gates + live MCP check.**
  Regression tests for the honest-reject boundary (a non-plane cap, a second picked edge at a far
  vertex → still floor via `cappingFaceAtFarVertex`; a spindle/clearing arm → still rejected by the arm
  builders). Consolidate the per-case watertight/fold/area assertions and the byte-identity pin into
  `model/feature/occtparity/curved_runout_test.go`. Live MCP-bridge test (`Oblikovati.AddIns.MCPBridge`):
  drive AddFillet+Recompute on one representative fixture (C5 or B6), screenshot-verify the rendered
  runout (fillet fades cleanly to the cap, no fold/hole), per the CLAUDE.md Live-tests discipline and
  the memory `validate-fillet-through-feature-not-ops`. **Gate:** corpus 72; the mutation checks fail if
  a real regression is introduced; live shot clean; full `go test ./kernel/ops/... ./model/feature/...`
  (only the not-yet-greened corpus reds remain — the simple green SET is exactly the prior 64 + the 8);
  `golangci-lint run ./kernel/ops/... ./model/feature/...` 0 issues.

**Order:** R1 (tracer, both arm kinds) → R2 (remaining perp) → R3 (oblique) → R4 (gates + live). R2
needs R1's dispatch+loop; R3 needs R1's loop to hang the oblique end on; R4 needs all 8 green. Each
task: fresh opus implementer + fable review + mesh-hash byte-identity + fold/watertight gates.

## Self-review

- **Coverage:** all 8 cases mapped to a task (R1: B6,C9; R2: M7,C1,C5,D8; R3: D4,E3; R4: gates). The
  forensic's P1–P5 collapse into R1 (P1+P2 tracer), R2 (P2 rest), R3 (P3), R4 (P4 gates + P5 rejects).
- **Type consistency:** `isSingleArmRunout`/`singleArmRunoutBody`/`closeSingleArmRunoutRails` named once
  and reused; `armFarRunout(ef, cornerWeld{center,radius}, h0, h1, filletedEdges, res)` matches the real
  signature (`fillet_far_runout.go`); `cornerWeld` fields `center`/`radius` verified as the only ones
  read.
- **No placeholders:** every task names its exact functions, cases, and numeric gates from the forensic
  DRAWEXE tables.
