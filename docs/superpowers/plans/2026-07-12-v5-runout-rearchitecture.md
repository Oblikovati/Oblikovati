# V5/V1 Runout Re-architecture — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.
> **Note:** Task 1 is a **spike** that resolves a load-bearing representation decision (arc-fit vs a new conic type). Do Task 1 and report its measurement BEFORE committing to Tasks 3–5's exact shape — the spike's outcome selects between the two branches in Task 3.

> **STATUS UPDATE (2026-07-12) — Task 1 DONE + design RESOLVED.** The spike (`.superpowers/sdd/task-1-report.md`) disconfirmed the shared-edge conic premise: neither arc-fit nor the exact `cyl∩plane` ellipse reaches <1% (V5 +1.24%, V1 +1.49% with the exact ellipse) — it closes only ~15% of the gap; 73% of the residual is on straight-edge-only flank triangles no conic can touch. A follow-up characterization (`.superpowers/sdd/v5-setback-characterization.md`) found and verified the true lever: **a rail-termination setback**. OCCT terminates each flank tangent rail where its generator **pierces the adjacent far plane** (`t_pierce = n·(Q−A)/(n·û)`, with `A`=rail point, `Q`=`fan.apex`, `n`=far-plane normal, `û`=`fan.axis`), not at the picked edge-vertex's axial projection (our `ta`/`tb`, which sit at the apex station `d>r` outside the tube). One line∩plane formula predicts all six setback vertices for **both** V5 and V1 to ≤3.4e-5·r. Predicted totals flank-fix-only: **V5 +0.39%, V1 +0.48%** (a lower bound; the setback also shrinks the cylinder and de-bulges the runout far faces). **Task 3 below is rewritten to this fix; the ellipse-edge chain is superseded** (an exact `cyl∩plane` ellipse edge is now only an optional 2nd-order weld-fidelity improvement, not a gate-closer).

**Goal:** Make `simple/V5` (valence-6) and `simple/V1` fillet runouts match OCCT's area within the 1% gate by re-architecting the runout rebuild to OCCT's construction: cylinder end-loop = a chain of shared `cyl∩far-plane` ellipse edges, far faces trimmed directly by those edges, no cap, apex vertex dropped.

**Architecture:** Replace the G5 first-slice cap + separate far-face arc-piece (`caps`/`capEndSegs` + `addRunoutApex`) with a direct cylinder↔far-face shared ellipse-edge trim, for the k=1 single-edge planar-far-face runout only. Ground truth + rationale: `docs/superpowers/specs/2026-07-12-v5-runout-rearchitecture-design.md`.

**Tech Stack:** Go, `go test`; kernel `kernel/ops` (`fillet_runout_rebuild.go`, `fillet_runout_spread.go`, `fillet_faces.go`), `kernel/geom`; corpus harness `model/feature/occtparity`; DRAWEXE oracle at `../occt-build/lin64/gcc/bin/DRAWEXE` (env `test-utilities/occt-blend/oracle/drawenv.sh`).

## Global Constraints
- Branch `feat/occt-blend-parity-corpus`; commit per task; NO PR (standing rule: no PR until all corpus cases pass).
- SPDX `GPL-2.0-only` line 1 on new .go. Functions 4–20 lines; explicit types; early returns; ≤2 indent; exception messages include offending value + expected shape; gofmt clean; model-scaled tolerances (no bare 1e-6).
- The pure solver `fillet_runout_spread.go` imports ONLY geom+math+fmt+stdmath.
- **Keep Step A** (near-apex root selection, `63ad9205`) — it lands runout vertices exactly on OCCT's.
- **Gate on k = (#filleted edges at V), not valence N.** k=1 → this construction; k≥2 → the existing `addCornerRound`/corner path, untouched.
- Regression: V5 AND V1 within 1% (PASS); V3 and X9 stay PASS; trihedral corpus byte-for-byte unmoved (stash-diff); no reopened shells (every edge used twice, `IsSolid`).

---

### Task 1 (SPIKE — gates the rest): does a no-cap trim with arc-fit reach <1%?
**Deliverable:** a measured answer to "with the cap deleted and far faces trimmed directly by the sub-arc chain, does a circular arc-fit of the small sub-arcs hit <1% for V5/V1, or is a true conic edge required?" No production commit required if it stays a throwaway spike; its OUTPUT is the decision.

- [ ] Build a throwaway spike (a `_test.go` you delete, or `experiments/`) that, for the real V5 fan, constructs the runout the OCCT way: cylinder end-loop = the ordered near-apex sub-arcs (Step-A splits) with NO cap face, far faces bounded directly by those same sub-arcs (V dropped), and measures the resulting per-face + total area two ways: (a) sub-arc as the current circular arc-fit; (b) sub-arc as the on-plane (true-ellipse) mid. Compare both totals to OCCT 24551.4.
- [ ] Report: which representation (if either) reaches <1% for V5 and V1 once the cap is gone. This decides Task 3's branch:
  - **arc-fit <1%** → Task 3 uses existing `geom.Arc3dByThreePoints`, no new primitive.
  - **only true-ellipse <1%** → Task 3 adds a minimal `geom` ellipse/rational-quadratic edge (its own sub-tasks) for `cyl∩plane`.
  - **neither <1%** → STOP and escalate; the residual is not the shared-edge representation and the design needs revisiting (do not proceed to a re-architecture that can't close the gate).
- [ ] Revert all spike instrumentation; record the measurement in the ledger. Commit nothing (or only a doc note).

---

### Task 2: Confirm the k=1 gate and lock the trihedral/k≥2 boundary
**Files:** `kernel/ops/fillet_runout_fan.go` (+ test).
**Deliverable:** the fan path provably only handles k=1 single-edge runouts; k≥2 corners stay on the existing path.

- [ ] Write a test asserting `classifyEndCorners` produces a fan ONLY for a corner where exactly one incident edge is filleted (k=1), and never for a k≥2 corner (which must remain blend/miter/`addCornerRound`). Use a real multi-pick fixture if one exists in the corpus; else a synthetic body with two filleted edges meeting at a vertex.
- [ ] If the current detector does not already exclude k≥2 (it skips blend/miter — verify that covers k≥2), add an explicit k=1 guard in the fan classifier. Keep it ≤20 lines.
- [ ] Run `go test ./kernel/ops/ -run 'TestClassifyEndCorners|TestVertexValence'`; commit.

---

### Task 3: Rail-termination setback — pierce the adjacent far plane, not the apex projection
**Files:** `kernel/ops/fillet_runout_fan.go` (compute pierce in `buildEndCornerFan`), `kernel/ops/fillet_faces.go` (rail endpoints in `cylinderFace`), possibly `kernel/ops/fillet_runout_rebuild.go`/`fillet.go` (threading the corrected endpoints) (+ tests).
**Design source:** `.superpowers/sdd/v5-setback-characterization.md` (the verified rule, the six target vertices, the predicted totals). **Do NOT build the disconfirmed ellipse-edge chain.**
**Deliverable:** for a pick edge with a runout (fan) end, each flank tangent rail terminates at its **pierce** of the adjacent far plane, and BOTH the cylinder rail and the fan's first/last cap-piece endpoint use that SAME pierce point (loop stays closed, weld-twice preserved). Result: V5 and V1 solids match OCCT area within the gate; runout interior (near-root splits) untouched.

The rule (verified to ≤3.4e-5·r on all six V5+V1 setback vertices): for a rail through point `A` (today's `ta`/`tb`) along `û = fan.axis`, terminate it at `A + t_pierce·û` where `t_pierce = n·(Q − A)/(n·û)`, `Q = fan.apex` (the runout vertex lies on every far plane incident to it), `n` = the adjacent far face's outward normal. Adjacent far plane: **rail A ↦ `fan[0].face`**, **rail B ↦ `fan[last].face`** at the fan (apex) end; **the single opposite far face** at a trihedral end of the same pick edge (V1's start end needs it too — it is also an apex `d>r`; apex-only leaves V1 at +1.10%, over).

**Coupling constraint (load-bearing — see `fillet_faces.go:313-316` + `fillet_runout_fan.go:120`):** `fan.ta`/`fan.tb` and the cylinder rail endpoints `ef.c1.ta`/`ef.c1.tb` both derive from the SAME `corner.ta`/`corner.tb`. The correction must be a SINGLE source (correct the corner's `ta`/`tb`, or thread the pierce points to both the rail seg and `buildEndCornerFan`), or the rail end and the first/last cap-piece endpoint disagree and the cylinder loop opens. Assert they coincide in a test.

**Scope guard (keeps the trihedral corpus byte-for-byte unmoved):** apply the setback ONLY on pick edges that have ≥1 fan (runout) end — the V-series fixtures only. Never change rail termination for a fillet with no runout end (that path is what Task 4's stash-diff must show unmoved). A trihedral end gets the setback only because the OTHER end of the same edge is a fan.

- [ ] **Verify V1's end classification first (cheap, informs the build):** a throwaway assertion that V1's pick edge has one fan (valence-4) end and one trihedral end (or two fans — confirm which), and that both end vertices sit at `d>r` from the cylinder axis. This decides whether the trihedral-end branch is exercised. Record the finding.
- [ ] **Write the failing test:** on the real V5 fan (reuse `solvedFilsForCase`/`filletPickForCase`), assert `buildEndCornerFan`'s corrected `fan.ta`/`fan.tb` equal the OCCT pierce vertices RV_3 (39.5842,88.1164,51.7052) and RV_12 (40.0319,85.8487,47.3618) within `1e-3` (≈2e-4·r). Prove RED against the current apex-projection `ta`/`tb`.
- [ ] **Implement the pierce** in `buildEndCornerFan` (it already orders the chain, so `fan[0]`/`fan[last]` and their normals are in hand; `apex`, `axis`, `radius` too): compute `ta`/`tb` as the line∩plane pierce. Guard `|n·û|` against a model-scaled floor (rail parallel to the far plane ⇒ no pierce ⇒ honest-reject via `filletRunoutError`). Green the test.
- [ ] **Thread the corrected endpoints to the cylinder rail:** ensure `cylinderFace`'s A-rail `to` / B-rail `from` at a fan corner use the SAME corrected pierce point as `boundaryPoint`'s flank rail (`fan.ta`/`fan.tb`) — the first/last `capEndSegs` piece endpoint. Add a test asserting the rail endpoint == the first/last cap-piece endpoint for the V5 fan (loop closed).
- [ ] **Trihedral end of a runout edge:** apply the same pierce (against the single opposite far face) at the trihedral end of a pick edge whose other end is a fan, so V1's start rails set back. Scope-guard it so a fillet with no fan end is untouched.
- [ ] **Weld/solid invariant:** on the real V5 AND V1 bodies, assert `IsSolid() && Validate().Valid`, every edge used exactly twice, apex vertex absent (unchanged from today), runout interior split points unchanged (RV_9/10/11 still equal `solveRunoutSpread`'s output — the fix must NOT move the interior).
- [ ] Run FULL `go test ./kernel/ops/`; `go build ./...`; commit.

---

### Task 4: Area gate — V5/V1 to PASS, V3/X9 stay PASS, trihedral unmoved
**Files:** `model/feature/occtparity/fillet_g5_runout_test.go` (+ scoreboard verification).
**Deliverable:** V5/V1 within 1%; the gate promotes them; the tripwire is retired.

- [ ] Measure V5/V1/V3/X9 areas + verdicts. Confirm V5 AND V1 now PASS (within 1%); V3/X9 still PASS.
- [ ] Update the gate: `TestG5RunoutCasesPass` gates V5 (and V1 if it is a corpus case) as PASS; DELETE/repurpose `TestG5V5StillFailsArea` (V5 is fixed — it should now assert PASS or be removed).
- [ ] Scoreboard: confirm total PASS rises by exactly {V5, V1} (was 29 → expect 31) with NO other case moved (worktree stash-diff at the pre-Step-A/pre-re-arch commit, as in the G5 slice's Task 8). If any other case flips, STOP and report.
- [ ] Full green + lint: `go test ./kernel/ops/ ./model/feature/occtparity/`, `go build ./...`, `go vet ./...`, `gofmt -l`, `golangci-lint run ./kernel/ops/... ./model/feature/occtparity/...`. Commit.

---

### Task 5: Reconcile docs
**Files:** the design spec, the roadmap.
- [ ] Update `docs/superpowers/specs/2026-07-11-occt-blend-greening-roadmap.md`: V5/V1 moved from FailArea to PASS; note the runout re-architecture landed; record any deferred items (quadric far faces, k≥2 corner blends).
- [ ] If Task 1 required a new `geom` conic type, add/adjust the relevant ADR. Commit.

## Self-Review checklist (run before starting Task 3)
- Task 1's measurement is recorded and its branch chosen (arc-fit vs conic).
- The k=1 gate (Task 2) is confirmed so the trihedral/k≥2 path is provably untouched.
- Every shared sub-arc edge is used exactly twice (cylinder + one far face) — the weld invariant.
- V3/X9 re-baselined onto the new path stay PASS; trihedral corpus unmoved (stash-diff).

## Open items to resolve during execution
1. Task 1 spike outcome selects Task 3's sub-arc representation (arc-fit vs new conic type) — a real dependency; do not pre-commit to adding a `geom` primitive before the spike says it is needed.
2. Whether the far-face "trim" is a pure loop-boundary change or needs surface pcurve handling in this kernel — design against the real `filletFace`/`filletLoop` code during Task 3.
