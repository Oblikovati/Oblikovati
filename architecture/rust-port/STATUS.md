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
- Scenario pin: **not established**. The live `/mnt/vininas-projects/MCAD-Scenarios` directory has no Git metadata. Do not call a file hash a commit.
- Source sync: weekly, in a separate batch. Store small JSON receipts in this repository.
- Kernel gaps: file and track validated gaps in `Oblikovati/Martensite`; application work belongs in `Oblikovati/Oblikovati`.
- Branch policy: one working branch, with batch PRs to `rust`; push after each unit once the required pre-push gate passes.
- The current `AGENTS.md` is local and untracked. A5 must reconcile and track the applicable rules.

## Queue

| Order | Action | State | Evidence / dependency |
| --- | --- | --- | --- |
| 1 | A5: reconcile agent instructions and the port plan | Ready | Preserve user edits; confirm tools and paths before documenting them. |
| 2 | A6: report all scenario binding and runtime gaps | Ready | Do not claim that steps blocked by failed setup ran. |
| 3 | A7: pin the scenario source | Ready | The live source has no Git history; preserve an exact snapshot and record its commit. |
| 4 | A8: generate the behavior ledger and progress counts | Ready | Issue #3861; requires the scenario pin and receipt mapping. |
| 5 | A9: map feature files to milestones | Ready | Issue #3861; validate mappings against the milestone exits. |

## Open items

- A2: blocked by the full gate. The run on `6c5c1014a` is still collecting test results; a repository preflight found 27 files at or above the 500-line limit. See issue #3860. A fresh gate is required before the first push.
- A3: stale worktrees and 24 ancestor branches removed. Two branches contain commits outside the working branch and remain for review; `unmerged-lanes-20261113.bundle` on the NAS backs up both. The `oblikovati-rust-backups` directory on the NAS holds the latest verified branch bundle.
- A4: this status file is the initial manual queue. Generate it from the ledger when A8 is ready.
- A1: appearance and material resolution committed at `718a5d545`; Body Properties passed 6/6 scenarios and 69/69 steps on the saved source snapshot. See `receipts/2026-11-13-body-properties/receipt.json` and issue #3859.
- Per-face appearance presentation is not established by the body-color work. Verify it separately.
- Do not mark R0–R3 closed without their exit checks and receipts.
