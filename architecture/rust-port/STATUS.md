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
- At `7075016f1`, the ledger reports **0 verified**, 158 stale, 36 blocked
  and 38 excluded across 2,720 declarations. Native build provenance is not
  bounded (#3978); prior captures remain historical evidence. The gate also
  requires missing baseline fields.
- `rust-work` is local and has not passed the full pre-push gate. All 20 former
  application PR heads were merged locally and their PRs closed. Do not remove
  their remaining remote branches until the consolidated work is safely pushed.
- The 2026-11-15 `make gate` stopped at the API pin check, before workspace
  tests: Express requires an unreleased API contract. Release and pin work is
  tracked in #3965. API branch `agent/express-release-prep` passed its gate and
  its issue-scope duplication scan at 1.0269%. The two example packages now
  report 95.7% and 85.7% coverage; aggregate Go coverage is 84.6%. The branch
  is local and no release was published. The ledger also rejects stale receipts.
- Scenario pin: `05317bbb2d0809a532b29b7ff6f726a5722b865f`, a local snapshot
  of the unversioned source. It is not an upstream commit. The live feature
  inventory matched the pin at the source sync. The source tags 38 observations
  as capture-only and the remaining 34 document-creation cases as capture-host
  only; these are excluded or blocked, not passing. Follow `bdd-workflow.md`
  when source files change; use a unique Cargo target for each active worktree.
- The native harness injects input into the window frame loop. It does not prove
  OS-level key and mouse delivery. Close test windows and retain receipts.
- `make docs-lint` passed after the source-size and worktree-discovery fixes
  (#3970, #3971). The full `make gate` remains open.
- Martensite work is owned separately. Report validated kernel gaps in
  `Oblikovati/Martensite`; do not modify the kernel in this lane.
- GUI unit tests isolate the shared ImGui context under parallel execution
  (#3878). The final eight-thread run passed 495 tests; the production context
  ownership check still rejects a second active owner. Native windows and
  headless diagnostics now share a test-only lease (#3878). Two Clear tests
  passed twice with two threads, and a mixed group of 15 passed with eight;
  the full native head suite remains unverified. A partial serial run
  recorded two failures; see `receipts/2026-11-15-head-partial/`. The native
  composition test passes with the sized Xvfb setup (#3973); use
  `make gate-xvfb` to run the actual gate on a suitable virtual display.

## Queue

| Order | Action | State | Tracking |
| --- | --- | --- | --- |
| 1 | Complete the behavior ledger and R0 baseline fields | In progress | #3865 |
| 2 | Review milestone mappings and Go baselines | In progress | #3861 |
| 3 | Replace old prose claims with current durable receipts | In progress | `behavior-receipts.md` |
| 4 | Audit R0–R3 exits and the full gate | In progress | #3860 |
| 5 | Bound native runner build provenance | In progress | #3978 |

## Active gaps

- Native captures cannot establish the runner build inputs (#3978). The
  experimental committed-source exporter checks tracked blobs but does not
  prove a build. Scenario dependencies cannot authorize receipt reuse (#3974).
- Sketch extrusion lacks a certified solve precision bound (#3928). Analytic
  edge picking is blocked by [Martensite #462](https://github.com/Oblikovati/Martensite/issues/462).
- The full gate needs a released API contract (#3965). The Point feature passed
  8 of 9 native scenarios; point-10 needs its recorded fixture (#3887).
- Rectangle visibility now rejects obstructed observations (#3980). Focused
  native checks passed; the full gate is still required.
- Authored work planes now display in part views (#3981). Viewport plane
  picking now supports sketch and offset-plane creation (#3984); assembly
  overlays and picking remain open. Visibility editing needs the
  document-qualified datum reference work (#3982, #3983). Tested drafts remain
  unmerged: Rust `work/datum-visible-3982` at `2c0601be1` and API
  `work/datum-reference-contract-3982` at `c053e98`. Public compatibility,
  Go host migration, origin visibility and axis reporting remain open.
- See [ACTIVE-GAPS.md](ACTIVE-GAPS.md) for individual scenario diagnostics,
  source clarifications, and remaining UI work. Those run results are historical.

## Evidence and history

`behavior-ledger.json` and its generated report are the current counting source.
`receipts/` contains native reports, captures and diagnostic runs. A failed or
instrumented run is not acceptance evidence. The previous detailed status is
preserved in [STATUS-history-2026-11-15.md](STATUS-history-2026-11-15.md); its
old pass claims are historical and can be stale.
