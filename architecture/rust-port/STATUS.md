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

- A2: the diagnostic gate started on `6c5c1014a` and failed with 36 head test failures. The source changed during the run, so it cannot certify the current HEAD. See issue #3860 and `receipts/2026-11-13-gate-diagnostic/`. The gate preflight reports 2,695 unclassified scenarios. A fresh passing gate is required before the first push.
- A3: stale worktrees and 24 ancestor branches removed. Two branches contain commits outside the working branch and remain for review; `unmerged-lanes-20261113.bundle` on the NAS backs up both. The `oblikovati-rust-backups` directory on the NAS holds the latest verified branch bundle.
- A8: stale application or source evidence is downgraded without hiding damaged artifacts (#3869). Ledger-tool-only edits do not invalidate native UI tests.
- A8: the ledger records 2,719 declarations, including 62 blocked by invalid Gherkin (#3866). Eight prior native receipts became stale when automation source moved; rerun them before counting them as current. Another 2,695 scenarios remain unclassified. See `behavior-receipts.md` and issue #3865.
- A8: `behavior_cli.py backlog` groups the 195 feature files by their assigned milestones and shows missing classifications and R0 evidence. All 2,719 rows still have incomplete R0 baseline fields; unassigned work is not given an inferred milestone.
- A10: the former prose ledger is archived in `archive/bdd-work-ledger-2026-11-13-*.md`. Its old temporary paths and pass labels are historical, not current evidence. Only the linked receipts can support current verified claims.
- R4 rectangle sizing: the first two-point rectangle scenario has a stale native receipt after the automation refactor; later rectangle scenarios remain unverified. See issue #3857.
- R4 extrusion: the `extrusion-01` native receipt is stale after the automation refactor. The full cube journey remains unverified (#3870).
- R3 cube evidence: the headless certified replay and the native GUI-loop cube test passed on `77f03cf61`; see `receipts/2026-11-14-r3-cube-check/`. The native test uses injected events and an API model-state fixture. Foreground MCPBridge input and screenshot evidence, full R3 exit checks, and the gate remain open (#3857).
- A7: the scenario bundle, file manifest, and verifier are tracked in `test-utilities/benchmarks/scenario-snapshots/` and `tools/bdd/`. See issue #3864.
- A6: diagnostic runner reports all steps and taints results after a failure. See issue #3863 and `receipts/2026-11-13-bdd-diagnostics/`. Fixture blockers require an explicit fixture signal.
- A4: this status file is the initial manual queue. Generate it from the ledger when A8 is ready.
- A5: applicable agent rules are tracked in `AGENTS.md` and its linked rule files. The branch and feature documentation policies match the port plan.
- A1: appearance and material resolution committed at `718a5d545`; Body Properties passed 6/6 scenarios and 69/69 steps on the saved source snapshot. See `receipts/2026-11-13-body-properties/receipt.json` and issue #3859.
- The stale green pixel oracle is replaced in the targeted native tests (#3867). A rotation test now reaches a separate constraint-policy question (#3868); the full gate has not been rerun.
- Historical native evidence: see `receipts/2026-11-14-body/`, `receipts/2026-11-14-rectangle/`, and `receipts/2026-11-14-extrusion/`; rerun after the automation refactor.
- A12: moved document lifecycle commands out of the 511-line file host and split the 509-line assembly contract tests (#3860). The next `docs-lint` failure is `crates/gui/src/browser.rs`.
- Per-face appearance presentation is not established by the body-color work. Verify it separately.
- Do not mark R0–R3 closed without their exit checks and receipts.
