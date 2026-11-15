# Rust port status

Last checked: 2026-11-15. Refresh from current evidence at each checkpoint.

## Start here

1. Read this file, `ACTION-PLAN.md`, the pending handoff, and `git status`.
2. Preserve the user's modified `README.md` and existing stashes.
3. Verify the pinned scenario checkout and receipt before relying on a result.
4. Use the Go application as a reference and the BDD scenarios as acceptance.
5. Merge finished work into local `rust-work`; clean merged task worktrees and caches.

## Current state

- **R0 remains open.** The full gate, baseline fields and milestone exit audits
  are incomplete. Run `python3 -B tools/bdd/behavior_cli.py report` for current
  counts; source and application changes can make earlier receipts stale.
- At `7fcec3bad870335d8877a7c4919dc5f67ffb4edc`, the ledger reported five
  current verified scenarios, 173 stale receipts and 21 blocked scenarios
  across 2,720 declarations. This is a checkpoint, not a permanent count.
  The gate also requires baseline fields for 2,580 scenarios.
- `rust-work` is local and has not passed the full pre-push gate. All 20 former
  application PR heads were merged locally and their PRs closed. Do not remove
  their remaining remote branches until the consolidated work is safely pushed.
- The 2026-11-15 `make gate` stopped at the API pin check, before workspace
  tests: Express requires an unreleased API contract. Release and pin work is
  tracked in #3965. The separate ledger check reports 178 stale receipts.
- Scenario pin: `48987af7b5cbb06515bc6f30385f508382f34064`, a local snapshot
  of the unversioned source. It is not an upstream commit. The live feature
  inventory matched the pin at the last source check. Follow `bdd-workflow.md`
  when source files change; use a unique Cargo target for each active worktree.
- The native harness injects input into the window frame loop. It does not prove
  OS-level key and mouse delivery. Close test windows and retain receipts.
- Martensite work is owned separately. Report validated kernel gaps in
  `Oblikovati/Martensite`; do not modify the kernel in this lane.
- GUI unit tests isolate the shared ImGui context under parallel execution
  (#3878). The final eight-thread run passed 495 tests; the production context
  ownership check still rejects a second active owner.

## Queue

| Order | Action | State | Tracking |
| --- | --- | --- | --- |
| 1 | Complete the behavior ledger and R0 baseline fields | In progress | #3865 |
| 2 | Review milestone mappings and Go baselines | In progress | #3861 |
| 3 | Replace old prose claims with current durable receipts | In progress | `behavior-receipts.md` |
| 4 | Audit R0–R3 exits and the full gate | In progress | #3860 |
| 5 | Split over-limit Rust files | Done | Guard remains enforced (#3860) |

## Active gaps

- Viewport orientation: all 20 pinned scenarios passed on the merged revision
  (#3961). See `receipts/2026-11-15-orientation-verified/`. The run uses
  frame-loop input; OS-level input and the full Go baseline remain open.

- Document opening: Express open and Load Full are in `rust-work` (#3949).
  The pinned journey passes 13 of 14 steps; Finish and Simplify need the
  scenario clarification recorded in #3963. Saved views require matching source,
  mesh quality and producer identity. Cache creation is unavailable on macOS
  and Windows until the running executable can be verified (#3964). See
  `receipts/2026-11-15-express-contract/` for the diagnostic run.
- Document creation: the full diagnostic passed 29 of 39 scenarios. Named
  template assets and their catalog configuration are missing (#3899, #3951,
  #3952). The scenario owner's `BLOCKERS.md` lists the fixture requirements.
- Document tabs: the diagnostic passed 9 of 38 scenarios (#3900). Two context
  menu scenarios passed on the merged revision; see
  `receipts/2026-11-15-tab-context-verified/`. Two Home menu scenarios need
  source clarification in their `BLOCKERS.md`. The keyboard scenario receipts
  predate later source changes (#3950).
- Quick access: the diagnostic passed 11 of 12 scenarios. Measure and Clear
  for one preselected body are available. Clear has confirmation, undo and
  redo checks in `receipts/2026-11-15-clear-appearance/`; its wider selection
  workflow remains open (#3962). The recorded profile remains incomplete
  (#3955).
- Body properties: all six pinned scenarios passed with checked setup, list
  scrolling and exact physical-value rows (#3954). Receipts become stale after
  relevant source changes; see `receipts/2026-11-15-body-properties-verified/`.
- Sketch ribbon: inactive-sketch setup was unbound (#3956). The source's
  outside-edit Create availability conflicts with the Go gate; the scenario
  owner has an actionable blocker.
- Ribbon appearance: 16 of 18 pinned scenarios passed; the Team Web fixture
  remains missing (#3957). See `receipts/2026-11-15-ribbon-appearance/`.
- Two-point rectangle: nine of ten scenarios passed in a configured diagnostic
  run. Scenario 10 needs the recorded fixture to assess its transient count
  (#3953). See the scenario owner's `BLOCKERS.md`.
- Martensite closest-point behavior is tracked separately in
  [Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).

## Evidence and history

`behavior-ledger.json` and its generated report are the current counting source.
`receipts/` contains native reports, captures and diagnostic runs. A failed or
instrumented run is not acceptance evidence. The previous detailed status is
preserved in [STATUS-history-2026-11-15.md](STATUS-history-2026-11-15.md); its
old pass claims are historical and can be stale.
