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
- At `433e83c4548590327ff2805238d6daaf066d41f5`, the ledger reported zero
  current verified scenarios, 156 stale receipts and 21 blocked scenarios
  across 2,720 declarations. This is a checkpoint, not a permanent count.
- `rust-work` is local and has not passed the full pre-push gate. All 20 former
  application PR heads were merged locally and their PRs closed. Do not remove
  their remaining remote branches until the consolidated work is safely pushed.
- Scenario pin: `48987af7b5cbb06515bc6f30385f508382f34064`, a local snapshot
  of the unversioned source. It is not an upstream commit. The live feature
  inventory matched the pin at the last source check. Follow `bdd-workflow.md`
  when source files change; use a unique Cargo target for each active worktree.
- The native harness injects input into the window frame loop. It does not prove
  OS-level key and mouse delivery. Close test windows and retain receipts.
- Martensite work is owned separately. Report validated kernel gaps in
  `Oblikovati/Martensite`; do not modify the kernel in this lane.

## Queue

| Order | Action | State | Tracking |
| --- | --- | --- | --- |
| 1 | Complete the behavior ledger and R0 baseline fields | In progress | #3865 |
| 2 | Review milestone mappings and Go baselines | In progress | #3861 |
| 3 | Replace old prose claims with current durable receipts | In progress | `behavior-receipts.md` |
| 4 | Audit R0–R3 exits and the full gate | In progress | #3860 |
| 5 | Split over-limit Rust files | Done | Guard remains enforced (#3860) |

## Active gaps

- Document opening: Express assembly loading, qualified cache and Load Full
  remain incomplete (#3949). A real saved assembly fixture and shared live
  presenter are in `rust-work`; cache and API work are in progress.
- Document creation: the full diagnostic passed 29 of 39 scenarios. Named
  template assets and their catalog configuration are missing (#3899, #3951,
  #3952). The scenario owner's `BLOCKERS.md` lists the fixture requirements.
- Document tabs: the diagnostic passed 9 of 38 scenarios (#3900). The four
  keyboard scenarios were bound and passed before later source changes (#3950).
- Quick access: the diagnostic passed 11 of 12 scenarios. Measure is available;
  other expected commands and the recorded profile remain incomplete (#3955).
- Body properties: all six pinned scenarios passed with checked setup, list
  scrolling and exact physical-value rows (#3954). Receipts become stale after
  relevant source changes; see `receipts/2026-11-15-body-properties-verified/`.
- Sketch ribbon: inactive-sketch setup was unbound (#3956). The source's
  outside-edit Create availability conflicts with the Go gate; the scenario
  owner has an actionable blocker.
- Ribbon appearance: seven of 18 diagnostic scenarios passed; unbound steps and
  the missing Team Web group remain (#3957).
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
