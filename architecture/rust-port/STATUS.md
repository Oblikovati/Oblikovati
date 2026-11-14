# Rust port status

Last checked: 2026-11-14. Update this file from current evidence at each checkpoint.

## Start here

1. Read this file, the pending handoff, and `git status`.
2. Finish or preserve uncommitted work before taking a new queue item.
3. Verify the scenario pin and receipt before relying on a scenario result.
4. Follow `ACTION-PLAN.md` for the work loop and `AGENTS.md` for project rules.

## Current state

- Milestone: **R0 open**. The behavior inventory and full gate are not yet verified.
- Work branch: `feat/rust-consolidated`; base is `origin/rust`.
- Scenario pin: `35f4a77b1519e2f29005be009c35fb7c5bdf4e89`, a locally created snapshot of the unversioned live source, not an upstream commit. Restore and verify it as described in `bdd-workflow.md`.
- Source sync: weekly, in a separate batch. Store small JSON receipts in this repository.
- Kernel gaps: file and track validated gaps in `Oblikovati/Martensite`; application work belongs in `Oblikovati/Oblikovati`.
- Branch policy: one working branch, with batch PRs to `rust`; push after each unit once the required pre-push gate passes.

## Queue

| Order | Action | State | Evidence / dependency |
| --- | --- | --- | --- |
| 1 | A8: complete the behavior ledger and progress counts | In progress | Issue #3865; reporting and receipt checks exist, but classifications, outline evidence, and R0 fields remain. |
| 2 | A9: map feature files to milestones | Ready | Issue #3861; validate mappings against the milestone exits. |
| 3 | A10: replace prose evidence with durable receipts | In progress | Historical prose is archived; scenarios without retained current reports still need reruns and receipts. |
| 4 | A11: record R0–R3 exit evidence or named gaps | Ready | Depends on the inventory and milestone mappings. |
| 5 | A12: split files over the line limit | Ready | Issue #3860; preserve behavior and run the required gate. |

## Open items

- A2: the diagnostic gate started on `6c5c1014a` and failed with 36 head test failures. The source changed during the run, so it cannot certify the current HEAD. See issue #3860 and `receipts/2026-11-13-gate-diagnostic/`. The latest gate preflight reported 2,610 unclassified scenarios; the current ledger has 2,592. A fresh passing gate is required before the first push.
- A3: stale worktrees and 24 ancestor branches removed. Two branches contain commits outside the working branch and remain for review; `unmerged-lanes-20261113.bundle` on the NAS backs up both. The `oblikovati-rust-backups` directory on the NAS holds the latest verified branch bundle.
- A8: stale application or source evidence is downgraded without hiding damaged artifacts (#3869). Ledger-tool-only edits do not invalidate native UI tests.
- A8: the ledger records 2,719 declarations, including 62 blocked by invalid Gherkin (#3866). Sixteen scenarios have current native receipts; 2,592 remain unclassified. Outline receipts require every expanded example and cleanup. See `behavior-receipts.md` and issue #3865.
- A8: `behavior_cli.py backlog` groups the 195 feature files by their assigned milestones and shows missing classifications, R0 rows, and counts for each missing baseline field. All 2,719 rows still have incomplete R0 baseline fields; unassigned work is not given an inferred milestone.
- A8: 15 native receipts were refreshed after the command-state split. Two initial runs hit an unavailable IBus input method and passed with `XMODIFIERS=@im=none`; some other captures show the warning. See `receipts/2026-11-14-rerun-environment/`.
- A8: the first body-properties row now records its Go reference, analytic oracle, kernel path, and platform limits. Go command tests pass but do not verify the body dialog or numeric result; its Go test premise remains open. See `receipts/2026-11-14-go-physical-baseline/`.
- A10: the former prose ledger is archived in `archive/bdd-work-ledger-2026-11-13-*.md`. Its old temporary paths and pass labels are historical, not current evidence. `tools/bdd/capture_cli.py` runs one reviewed declaration and prepares a checked receipt for visual review. Only linked, reviewed receipts support current verified claims.
- R7 orbit: Home fits into the document area while preserving the orbit pivot (#3872). Both examples of the reverse-rotation outline pass with visibly distinct faces; see `receipts/2026-11-14-generated-orbit/`. The viewport uses Martensite lighting and retains per-triangle colors. Curved surface normals and the full visual-style and lighting controls remain open (#3873).
- R7 ribbon docking: the top-to-left-to-right-to-top scenario passes all 18 steps. See `receipts/2026-11-14-command-rerun-0db0d6b60a17036c/`; separate diagnostic captures of both side layouts are in `receipts/2026-11-14-side-refactor/`. Other docking scenarios remain unverified (#3856, #3860).
- R4 rectangle sizing: the first two-point rectangle has a current native receipt; later rectangle scenarios remain unverified. See issue #3857.
- R4 new sketch: origin-plane creation, both Escape cases, YZ preselection, and distinct creation/Finish undo and redo steps have current native receipts (#3877, #3879, #3880). The origin-plane receipt is in `receipts/2026-11-14-command-state-sketch/`; the other current receipts are linked in the ledger. Five other declarations in the feature remain unbound. Finish is a transient history step and does not dirty the saved recipe. The Go reference discards that no-op step; the current BDD scenario specifies separate steps. The XZ Top fixture starts from above the plane because Go retains the viewer's side. Foreground captures are in `receipts/2026-11-14-viewcube-live/` and `receipts/2026-11-14-preselected-live/`.
- R5 face-hosted sketch: the pinned `new-sketch-07` diagnostic has five unbound steps; its later assertion is tainted. Stable face placement, replay, and native selection are tracked in #3884. See `receipts/2026-11-14-sketch-face-diagnostic/`.
- A9: all 43 viewport orientation, pan, zoom, and navigation-wheel declarations are assigned to R7 and remain inventoried (#3881). The four files match the live source as of 2026-11-14.
- A9: all 18 ribbon key-tip and nine viewport-orbit declarations are assigned to R7 (#3856, #3872, #3873). Two have current native receipts; the rest are inventoried, not verified. Both source files match the live source as of 2026-11-14.
- A9: all ten two-point rectangle scenarios have reviewed milestone assignments. The measurement and dimension-edit cases belong to R5; the remaining input cases belong to R4 (#3855, #3857).
- R4 extrusion: `extrusion-01` has a current native receipt. The full cube journey remains unverified (#3870).
- A9: all 30 extrusion scenarios have reviewed R4, R5, or R7 assignments. Only extrusion-01 has a current native acceptance receipt; the others remain inventoried until acceptance passes (#3857, #3870).
- R2 built-in parameter persistence: schema 3 retains the model-parameter guard and rejects invalid or older-schema flags (#3883).
- R2 save evidence: 395 persistence tests and nine session save tests passed on Linux; see `receipts/2026-11-14-r2-save/` (#3592). This does not establish cross-platform replacement or the full R2 exit. The full workspace gate remains open.
- Kernel/profile mapping: Go drops a small square that Martensite admits, so one Martensite region alone does not verify a Go profile index. A translated square also fails Martensite area certification at the current pin and local main `4b76a823`. Diagnostic sources and results are in `receipts/2026-11-14-profile-parity-diagnostic/`; Martensite #446 is marked high priority.
- R3 legacy replay gap: nonempty Go sketch and feature records are refused by the package restore path; the existing cube replay uses a Rust recipe. A Go-generated constrained cube and externally captured face keys are in `receipts/2026-11-14-go-cube-fixture/`. Go saves the evaluated extrusion distance, so after reopen a size edit changes only the sketch dimensions. An isolated, strict candidate converter now covers point/line sketches, fixed, coincident, horizontal and vertical relations, and distance dimensions; it does not migrate or open a document. See `receipts/2026-11-14-legacy-sketch-candidate/`. A staged new-body distance extrusion uses an injected, Go-compatible profile resolver; the Go cube is checked through a fixture-only resolver and Martensite. See `receipts/2026-11-14-legacy-part-candidate/`. Go side-face witnesses for the original and a reordered fixture resolve to source-named Rust faces through edit, undo/redo, and Rust recipe replay; see `receipts/2026-11-14-go-face-lineage/`. The witnesses are external. Production profile mapping, package migration, other feature and sketch content, and in-document dependent-face replay remain open (#3882).
- R3 cube evidence: the headless certified replay passed on `77f03cf61`; see `receipts/2026-11-14-r3-cube-check/`. The native GUI-loop cube test passed on `be2d5caf9`; see `receipts/2026-11-14-keyboard-cube-check/`. A foreground X11 mouse/keyboard journey on `0e0500c33` creates, edits, saves and reopens the cube, then resolves a saved face key through MCPBridge. See `receipts/2026-11-14-foreground-cube/`. The full R3 exit checks and gate remain open (#3857).
- MCPBridge measures a saved face key after cube replay; the 20 mm cube face reports 400 mm². See `receipts/2026-11-14-face-area-live/`. Other `analysis.measure` types return explicit declines and remain open (#3876).
- A7: the scenario bundle, file manifest, and verifier are tracked in `test-utilities/benchmarks/scenario-snapshots/` and `tools/bdd/`. See issue #3864.
- A6: diagnostic runner reports all steps and taints results after a failure. See issue #3863 and `receipts/2026-11-13-bdd-diagnostics/`. Fixture blockers require an explicit fixture signal.
- A4: this status file is the initial manual queue. Generate it from the ledger when A8 is ready.
- A5: applicable agent rules are tracked in `AGENTS.md` and its linked rule files. The branch and feature documentation policies match the port plan.
- A1: appearance and material resolution committed at `718a5d545`; Body Properties passed 6/6 scenarios and 69/69 steps on the saved source snapshot. See `receipts/2026-11-13-body-properties/receipt.json` and issue #3859.
- The stale green pixel oracle is replaced in the targeted native tests (#3867). The rotation test now checks that a move of the fixed profile is refused without changing the sketch, history generation, or displayed solid (#3868).
- The native GUI save/undo test used obsolete toolbar coordinates on clean base `e01e3a6e9` (#3874); see `receipts/2026-11-14-notice-baseline/`. It now targets rendered controls and passes through the camera assertions. A separate native test checks that the first toolbar navigation arrow stays out of the viewport.
- Earlier native evidence: see `receipts/2026-11-14-side-body/`, `receipts/2026-11-14-side-rectangle/`, `receipts/2026-11-14-side-extrusion/`, `receipts/2026-11-14-side-orbit/`, `receipts/2026-11-14-side-keytips/`, and the `side-01` through `side-05` receipt directories. The foreground keyboard and mouse history run is in `receipts/2026-11-14-sketch-history-live/`. The 195 live feature files match the saved snapshot as of 2026-11-14.
- On clean base `016ff28dc`, the typed-prism test expected `InvalidGeometry` for a short curve. The current kernel reports `InvalidInput` at admission with a `DegenerateCurve` cause; the adapter and native tests now check that cause, API reporting, and recovery (#3875). See `receipts/2026-11-14-profile-decline/`.
- A12: split the file host, assembly contract tests, GUI frame engine, and browser snapshot validation (#3860). The side ribbon code is split from `chrome.rs`, and command state is split from `commands.rs`. The next `docs-lint` failure is `crates/gui/src/ribbon_panels.rs`. See `receipts/2026-11-14-command-state-refactor/`.
- GUI unit tests pass serially (371/371); parallel execution can fail on simultaneous native context ownership (#3878).
- Per-face appearance presentation is not established by the body-color work. Verify it separately.
- Do not mark R0–R3 closed without their exit checks and receipts.
