# Rust port status

Last checked: 2026-11-13. Update this file from current evidence at each checkpoint.

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
| 1 | A8: generate the behavior ledger and progress counts | In progress | Issue #3865; source inventory exists, but mappings, receipt reconciliation, and coverage counts remain. |
| 2 | A9: map feature files to milestones | Ready | Issue #3861; validate mappings against the milestone exits. |
| 3 | A10: replace prose evidence with durable receipts | Ready | Keep historical records distinct from verified evidence. |
| 4 | A11: record R0–R3 exit evidence or named gaps | Ready | Depends on the inventory and milestone mappings. |
| 5 | A12: split files over the line limit | Ready | Issue #3860; preserve behavior and run the required gate. |

## Open items

- A2: the diagnostic gate started on `6c5c1014a` and failed with 36 head test failures. The source changed during the run, so it cannot certify the current HEAD. See issue #3860 and `receipts/2026-11-13-gate-diagnostic/`. A fresh gate is required before the first push.
- A3: stale worktrees and 24 ancestor branches removed. Two branches contain commits outside the working branch and remain for review; `unmerged-lanes-20261113.bundle` on the NAS backs up both. The `oblikovati-rust-backups` directory on the NAS holds the latest verified branch bundle.
- A8: `behavior-source-inventory.json` records 2,719 declarations at the pinned source commit; 62 are blocked by invalid Gherkin (#3866). It is an inventory, not verified coverage.
- A7: the scenario bundle, file manifest, and verifier are tracked in `test-utilities/benchmarks/scenario-snapshots/` and `tools/bdd/`. See issue #3864.
- A6: diagnostic runner reports all steps and taints results after a failure. See issue #3863 and `receipts/2026-11-13-bdd-diagnostics/`. Fixture blockers require an explicit fixture signal.
- A4: this status file is the initial manual queue. Generate it from the ledger when A8 is ready.
- A5: applicable agent rules are tracked in `AGENTS.md` and its linked rule files. The branch and feature documentation policies match the port plan.
- A1: appearance and material resolution committed at `718a5d545`; Body Properties passed 6/6 scenarios and 69/69 steps on the saved source snapshot. See `receipts/2026-11-13-body-properties/receipt.json` and issue #3859.
- The stale green pixel oracle is replaced in the targeted native tests (#3867). A rotation test now reaches a separate constraint-policy question (#3868); the full gate has not been rerun.
- Per-face appearance presentation is not established by the body-color work. Verify it separately.
- Do not mark R0–R3 closed without their exit checks and receipts.
