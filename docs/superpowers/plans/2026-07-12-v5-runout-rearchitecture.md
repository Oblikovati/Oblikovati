# V5/V1 Runout Re-architecture — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.
> **Note:** Task 1 is a **spike** that resolves a load-bearing representation decision (arc-fit vs a new conic type). Do Task 1 and report its measurement BEFORE committing to Tasks 3–5's exact shape — the spike's outcome selects between the two branches in Task 3.

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

### Task 3: Rebuild — cylinder↔far-face shared ellipse-edge trim, no cap, drop V
**Files:** `kernel/ops/fillet_runout_rebuild.go`, `kernel/ops/fillet_faces.go`, `kernel/ops/fillet_runout_spread.go` (+ tests). **Branch on Task 1's outcome for the sub-arc representation.**
**Deliverable:** for a k=1 fan, the cylinder end-loop is the ordered shared sub-arc chain, each sub-arc shared once with its trimmed far face; no cap; V absent from the loops.

- [ ] **Far-face trim:** replace `addRunoutApex`'s "arc piece replaces apex" with a trim: the far face's loop drops the apex vertex and is bounded by `q_entry → [shared sub-arc] → q_exit` (the same curve object the cylinder uses). Ensure the far face stays planar (surface unchanged) and only its boundary loop changes.
- [ ] **Cylinder end-loop:** keep `cornerEndSegs`→`capEndSegs` producing the ordered sub-arc chain, but ensure each segment's curve is the SAME geometric object shared with the far face (so the edge welds cylinder↔far-face, used exactly twice) — remove any separate cap face/surface.
- [ ] **Drop V:** ensure the apex vertex is not emitted in any fan face loop nor as a standalone; every un-filleted edge at V remains.
- [ ] **Delete the cap path** for the fan case (the `caps` map / cap tiling) — the cylinder end IS the shared chain. Keep the trihedral `cornerEndSegs` arc branch untouched.
- [ ] TDD: on the real V5 body, assert the result closes (every edge used exactly twice, `IsSolid`), the apex vertex is absent, and each sub-arc edge is shared by exactly the cylinder and one far face. Prove RED (current cap structure) → GREEN.
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
