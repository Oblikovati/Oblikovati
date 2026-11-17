# Rust port status

Last checked: 2026-11-16 on `rust-work`. Refresh from current evidence at each checkpoint.

## Start here

1. Read this file, `ACTION-PLAN.md`, the pending handoff, and `git status`.
2. Preserve the user's modified `README.md` and existing stashes.
3. Verify the pinned scenario checkout and receipt before relying on a result.
4. Use the Go application as a reference and the BDD scenarios as acceptance.
5. Merge finished work into local `rust-work`; clean merged task worktrees and caches.

## Current state

- Parallel constraint hover (#4031), Ready-state line selection (#4033), and
  tool candidate hover (#4034) have X11 diagnostics on live source. Fixture and
  reference-capture gaps prevent acceptance; other mouse scenarios remain open.
- All five solid viewport interaction scenarios pass X11 diagnostics for
  extrusion and Fillet (#4038, #4045). The live source is not pinned and its
  fixture and capture tags remain; shading is under review (#4044). See
  `receipts/2026-11-17-viewport-fillet-bdd-4045/`.
- Named-edge Fillet creation, replay, undo/redo and save/reopen have focused
  API and native-window tests (#4045). The typed `features.edit` path supports
  certified radius edits. The desktop edit panel has a native input diagnostic;
  creation can also start before an edge is picked, and its preview retains
  selected input-edge feedback. Full BDD acceptance remains open. See
  `receipts/2026-11-16-fillet-history-4045/`.
- Solid Chamfer is not yet available in the Rust UI (#4046). The pinned
  Martensite builder passed detached-cube geometry and face-history probes
  on all 12 edges. Naming, replay, host operations and UI remain open; see
  `receipts/2026-11-17-chamfer-prism-probe-4046/`.
- Pointer-driven Line creation and preview cancellation have an X11 diagnostic
  (#4037). Move preview and cancellation also pass a focused X11 diagnostic
  (#4040). Overlap type cues and Select Other also pass a native diagnostic
  (#4041, #4042). Ready-state point and line hover also have native mouse
  diagnostics (#4043). Rectangle selection, context-menu dismissal and clearing
  also have an X11 diagnostic (#4033); reference fixture and capture gaps remain.
- The 60 fps sketch target is not met on the measured GPU. A 4-segment
  pointer-motion run measured 20.738 ms median and 24.861 ms p95 at 1440 × 900;
  the 60 fps frame budget is 16.67 ms. See `receipts/2026-11-16-sketch-frame-baseline-4031/`.
  Host capture is #4032; context picks now use checked projection batches (#4035).
  No-readback presentation is Martensite #501; efficient previews are #504.
  Projection costs: `receipts/2026-11-16-pick-phases-4035/` (#505); no input-to-display timing was measured.
  A separate 200-line redraw and capture probe measured 25.963 ms median and
  27.920 ms p95 for the combined work; see
  `receipts/2026-11-16-sketch-redraw-capture-4031/`. These are separate
  measurements, not an end-to-end latency result.
  Constraint glyphs now use a cached same-plane projection batch. A debug
  projection-only diagnostic and native hover run are recorded in
  `receipts/2026-11-16-glyph-projection-4035/`; 60 fps remains unproven.
- Application closing (#4030) has a native close guard and save prompt. The
  current live feature passed 3/3 scenarios in an X11 diagnostic at `2dee60eef`;
  see `receipts/2026-11-16-application-closing-4030/`. It is not in the pinned
  snapshot, two scenarios have reference capture gaps, and #3978 prevents a
  verified acceptance claim. A wider GUI run also found the clean-base #3850
  UI-scale test failure.
- Martensite v0.13.1 is pinned at `bef25f47548431c8f128f08f66a1f719bcee4982`
  (#4025). The adapter and native cube checks passed; the curved profile
  comparison still fails as reported in Martensite #407. See
  `receipts/2026-11-16-martensite-0131-4025/`.
- **R0 remains open.** The full gate, baseline fields and milestone exit audits
  are incomplete. Run `python3 -B tools/bdd/behavior_cli.py report` for current counts.
- Opening an already open part passed native keyboard and mouse diagnostics
  (#3889); see `receipts/2026-11-16-open-existing-native-3889/` for input limits.
- At local snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`, the ledger
  reports **0 verified** across 2,760 declarations. There are 0 stale
  classifications and 2,743 incomplete baselines (#3865, #4039). Baselines
  without a reviewed source hash now count as incomplete. Sketch3D has 10
  capture-only cases and no native runner; provenance is open (#3978).
- Startup sketches on XY, YZ and XZ are integrated (#4009). The full pinned
  New Sketch diagnostic passed nine scenarios and found one unbound; see
  `receipts/2026-11-16-new-sketch-current-18d90af/`.
  The zero-radius circle OS-input check passed (#4012); see `receipts/2026-11-16-circle-os-4012/`.
  These are diagnostics and supporting observations, not verified acceptance.
- `rust-work` is local and has not passed the full pre-push gate. All 20 former
  application PR heads were merged locally and their PRs closed. Do not remove
  their remaining remote branches until the consolidated work is safely pushed.
- The full gate needs a released API contract (#3965). The combined local API
  candidate is `a87a7fc` on `work/api-release-3965`; no release or hosted CI
  result exists for it. The full-scope duplication scan reports 1.0389%.
  API #304 still needs the scenario owner's Move policy clarifications. The
  next release must also include the user-approved driving-dimension behavior
  for existing sketch creation calls (#187).
- The #187 host implementations remain unmerged pending the released API pin:
  Rust `work/host-expressions-187` at `3b77b8dbc`, and Go
  `work/go-associative-187` at `1649e3cad`. Their focused and affected-package
  suites passed against the draft API. The Rust adapter's full suite fails on
  the clean-base Martensite #406 ellipse test. The Go branch's normal hook
  passed; its full pre-push gate has not run. The Go branch includes the #4027
  cache race fix; concurrent sketch mutation (#4028) and stale picks after
  in-place geometry edits (#4029) remain separate open defects.
- Scenario pin: `b576445b98cffd686dcc05aa8c017de2a591c8c8`, a local snapshot
  of the unversioned source. It is not an upstream commit. At the latest check,
  the live solid viewport and Sketch3D arc and lifecycle features had changed.
  Five Sketch3D features were added: curve on face, intersection curve, project
  to surface, silhouette curve, and spline (#4038). Review them in the next
  sync; the comparison covers `.feature` files only.
  The source tags 60 declarations as capture-only and 34 more as capture-host
  only; they are not portable passing evidence. Follow `bdd-workflow.md`
  when source files change; use a unique Cargo target for each active worktree.
- Wheel zoom passes a native mouse diagnostic (#3881); see
  `receipts/2026-11-16-native-wheel-3881/`. Check input paths; close test windows.
- `make docs-lint` passed after splitting the source files that exceeded the
  500-line limit following the Fillet integration. The full `make gate`
  remains open.
- Martensite is owned separately; report kernel gaps without changing it here.
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
  sandbox probe compared committed source, 280 registry packages and 22,972
  Git-sourced files in one build. Generated Git manifests and host inputs remain
  unverified. See `receipts/2026-11-15-combined-build-probe/`. The observation
  does not prove the build. Capture now builds from a commit export with a fresh
  target, an isolated Cargo home and offline caches. Toolchain binaries, cached
  dependencies and host inputs remain unbounded. The inventory preflight now
  also builds from a commit export with an isolated Cargo home and target; its
  toolchain and host inputs remain unbounded.
  See `receipts/2026-11-15-provenance-config-3978-diagnostic/`. Receipt reuse
  is also open (#3974). The experimental sandbox now inventories an explicit
  system tree before and after the build; the tree is not authenticated or
  immutable, so the probe still cannot authorize a receipt.
- Sketch extrusion lacks a certified solve precision bound (#3928). Analytic
  edge picking is blocked by [Martensite #462](https://github.com/Oblikovati/Martensite/issues/462),
  which is marked high priority. The unmerged curvature branch
  `work/sketch-curvature-3995` is blocked by the high-priority conic extrema
  issue [Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).
- The full gate needs a released API contract (#3965). The Point feature passed
  8 of 9 native scenarios; point-10 is excluded until its recorded fixture is supplied (#3887).
- Rectangle visibility now rejects obstructed observations (#3980). Focused
  native checks passed; the full gate is still required.
- Move and Measure now show transient sketch candidates (#4034). A native
  Measure scenario passes with mouse input; its unpinned fixture makes this
  diagnostic evidence. Sketch frame time still exceeds the 60 fps budget
  (#4032, #4035 and Martensite #501).
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
