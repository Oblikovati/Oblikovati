# Rust port status

Last checked: 2026-11-14. Update this file from current evidence at each checkpoint.

## Start here

1. Read this file, the pending handoff, and `git status`.
2. Finish or preserve uncommitted work before taking a new queue item.
3. Verify the scenario pin and receipt before relying on a scenario result.
4. Follow `ACTION-PLAN.md` for the work loop and `AGENTS.md` for project rules.

## Current state

- Milestone: **R0 open**. The behavior inventory and full gate are not yet verified.
- Work branch: local `rust-work`; base is `origin/rust`. The consolidated branch,
  Martensite 0.6 handoff, and two dependency PR heads are merged locally. The
  `feat/sketch-circle-extrude` was merged at `3ad8a7384`. All 20 open PR heads
  are ancestors of that local commit. Older desktop tests now use the current shell and the desktop test
  target compiles. Focused Xvfb tests passed for document ribbons, a mouse-drawn
  constrained circle extrusion, and scaled parameter entry. The full native
  cube journey passed after the local model-state fix for #3892. A later review
  found that duplicate extrusion routes shadowed certified validation (#3893).
  The local fix unifies extrusion and suppression on certified transactions;
  the native cube journey and 352 desktop owner tests pass with that fix.
  The remaining integration checks and full gate must pass before the branch
  is pushed.
  Old and new sketch recipe version 13 meanings have been reconciled using version
  17 for new spline-handle recipes. Model, persistence, and session suites passed
  1,640, 412, and 492 tests; the Martensite adapter passed 85 unit tests, and
  automation library and query tests passed 797. Prism, sketch-prism, and revolve
  integration groups passed 78 tests. The wider Martensite suite fails
  `planar_pick::rotated_ellipse_pick_matches_the_independent_occt_minimum`: it
  returns a boundary point instead of the expected closest point. This matches
  [Martensite #406](https://github.com/Oblikovati/Martensite/issues/406). The
  desktop library compiles; the richer parameter panel passes 58 focused tests,
  and its revised keyboard journey passes on both configured platforms. The GUI
  unit suite passed 460 tests, and the dockable-window guard passed after the
  unused older shell was removed. Its sketch options now use the current shared
  preference owner and passed a native mouse test. The parameter-view suite
  passed 38 tests after the local fix for #3891; the issue remains open until
  the fix lands. A full desktop unit diagnostic was stopped after failures and
  later source changes made it stale; it cannot certify the merge.
  The selection filter and history browser have focused GUI and head integration
  coverage. Selection priority is still not applied to competing picks (#3890).
  Remaining integration tests and the full gate still need validation. All 20
  PRs were closed after their heads were checked against the local merge. Nineteen
  remote branches remain; the closed npm Dependabot branch was removed automatically,
  but its exact commit remains available via `refs/pull/3791/head` and in the merge.
  Preserve the user README edit and the three existing stashes.
- Scenario pin: `48987af7b5cbb06515bc6f30385f508382f34064`, a locally created snapshot of the unversioned live source, not an upstream commit. Restore and verify it as described in `bdd-workflow.md`.
- Source sync: weekly, in a separate batch. Store small JSON receipts in this repository.
- The 2026-11-14 source sync includes reviewed corrections in `home-recent-documents`, `arc-center-point`, `move`, `coincident`, and `rectangle-two-point`. All 2,720 declarations parse; the previous pin's native receipts are stale until rerun. Coincident 07 now checks a shared point without choosing which point moves. Rectangle 13 checks the recorded fixture and measured result without requiring a stationary edge.
- The circular Arc 09 fixture that derived its angle from the expected length has been removed. The current pin provides screen coordinates and window size, but the captured sketch and view state is unavailable; its first step is unbound and later diagnostic results are tainted. See `receipts/2026-11-14-arc-fixture-premise/summary.json` (#3910). Do not count the scenario as verified until its premise is checked.
- Kernel gaps: file and track validated gaps in `Oblikovati/Martensite`; application work belongs in `Oblikovati/Oblikovati`.
- Branch policy: merge completed work branches into `rust-work`; run the required pre-push gate before pushing.

## Queue

| Order | Action | State | Evidence / dependency |
| --- | --- | --- | --- |
| 1 | A8: complete the behavior ledger and progress counts | In progress | Issue #3865; reporting and receipt checks exist, but classifications, outline evidence, and R0 fields remain. |
| 2 | A9: map feature files to milestones | In progress | Issue #3861; all declarations have initial mappings, but their Go baselines and milestone exits still need review. |
| 3 | A10: replace prose evidence with durable receipts | In progress | Historical prose is archived; scenarios without retained current reports still need reruns and receipts. |
| 4 | A11: record R0–R3 exit evidence or named gaps | Ready | Depends on the inventory and milestone mappings. |
| 5 | A12: split files over the line limit | Ready | Issue #3860; preserve behavior and run the required gate. |

## Open items

- A2: the diagnostic gate started on `6c5c1014a` and failed with 36 head test failures. The source changed during the run, so it cannot certify the current HEAD. See issue #3860 and `receipts/2026-11-13-gate-diagnostic/`. The latest gate preflight reported 2,457 unclassified scenarios; the current ledger has 1,839 after all application declarations were mapped (#3856, #3894–#3900). A fresh passing gate is required before the first push.
- A3: stale worktrees and merged local branches were removed after ancestry checks. Only `rust-work` and the unrelated `go-legacy` local branch remain. `unmerged-lanes-20261113.bundle` and the prior verified backup remain on the NAS.
- A8: stale application or source evidence is downgraded without hiding damaged artifacts (#3869). Ledger-tool-only edits do not invalidate native UI tests.
- A8: the ledger records 2,720 mapped declarations with no parser-blocked scenarios after the source correction (#3866). All eight pinned project-dialog scenarios passed on an earlier native input revision; their receipts are stale after the chooser test changes. They cover inspection and cancellation; finishing a new project remains disabled and the library lists are empty. The chooser test changes and forward-ray fix make 60 earlier receipts stale under the native-input policy; rerun them before counting them as current. The baseline also has 2,700 declarations without complete R0 fields; these require individual review. The key-tip and browser-selection runs check the intermediate states, and their final captures show the expected cleared states. Outline receipts require every expanded example and cleanup. See `behavior-receipts.md` and issue #3865.
- R7 File menu: all eight pinned scenarios and 59 steps passed on the committed merge. Each scenario has an individual receipt and an inspected capture; the Go baseline still needs review. See `receipts/2026-11-14-file-menu-f0b61ad2/` and the other `2026-11-14-file-menu-*` receipt folders, and #3894.
- R7 project dialog: the pinned diagnostic reported one passing declaration and seven with unbound steps; subsequent steps are tainted. See `receipts/2026-11-14-project-dialog-merge-diagnostic/` and #3895.
- A8: `behavior_cli.py backlog` groups the 195 feature files by their assigned milestones and shows missing classifications, R0 rows, and counts for each missing baseline field. 2,713 rows still have incomplete R0 baseline fields; unassigned work is not given an inferred milestone.
- A8: 15 native receipts were refreshed after the command-state split. Two initial runs hit an unavailable IBus input method and passed with `XMODIFIERS=@im=none`; some other captures show the warning. See `receipts/2026-11-14-rerun-environment/`.
- A8: the first body-properties row now records its Go reference, analytic oracle, kernel path, and platform limits. Go command tests pass but do not verify the body dialog or numeric result; its Go test premise remains open. See `receipts/2026-11-14-go-physical-baseline/`.
- A10: the former prose ledger is archived in `archive/bdd-work-ledger-2026-11-13-*.md`. Its old temporary paths and pass labels are historical, not current evidence. `tools/bdd/capture_cli.py` runs one reviewed declaration and prepares a checked receipt for visual review. Only linked, reviewed receipts support current verified claims.
- R7 orbit: Home fits into the document area while preserving the orbit pivot (#3872). Both examples of the reverse-rotation outline passed on the current inputs; see `receipts/2026-11-14-refresh-orbit-59bbc715/`. The viewport uses Martensite lighting and retains per-triangle colors. Curved surface normals and the full visual-style and lighting controls remain open (#3873).
- R7 wheel opacity: the apply-and-restore scenario passed all 11 steps in the native GUI loop; see `receipts/2026-11-14-head-batch-d18b6850de15454e/` (#3881).
- R7 model browser: selecting an extrusion, its edge highlight and toolbar, then clearing the selection passed in the native GUI loop. See `receipts/2026-11-14-outline-9a6cb2d0318b1fcb/`.
- R7 ribbon docking: the top-to-left-to-right-to-top scenario passed all 18 steps after the panel catalog split. See `receipts/2026-11-14-ribbon-split-0db0d6b60a17036c/`; separate diagnostic captures of both side layouts are in `receipts/2026-11-14-side-refactor/`. Other docking scenarios remain unverified (#3856, #3860).
- R4 rectangle sizing: scenarios 01, 06–09, 11, 12 and 14 passed individually on the previous pin in the native GUI loop on Linux with Vulkan Lavapipe under Xvfb; those receipts are stale. Typed sizes, edge lengths, cancellation, undo and redo are checked. Scenario 10 still lacks the recorded transient dimension count. Corrected scenario 13 passed all ten steps on the current pin; see `receipts/2026-11-14-corrected-rectangle-13/`. See `receipts/2026-11-14-rectangle-gaps/` and issues #3857 and #3886.
- R4 new sketch: origin-plane creation, both Escape cases, YZ preselection, and distinct creation/Finish undo and redo steps passed on current code; the Go command baseline and its limits are in `receipts/2026-11-14-newsketch-go-baseline/` (#3877, #3879, #3880). The current native receipts are linked in the ledger; older captures remain historical. Five other declarations in the feature remain unbound. Finish is a transient history step and does not dirty the saved recipe. The Go reference discards that no-op step; the current BDD scenario specifies separate steps. The XZ Top fixture starts from above the plane because Go retains the viewer's side. Foreground captures are in `receipts/2026-11-14-viewcube-live/` and `receipts/2026-11-14-preselected-live/`.
- R5 face-hosted sketch: the pinned `new-sketch-07` diagnostic has five unbound steps; its later assertion is tainted. The model can resolve, project, and initially solve against part datums, with a session generation guard; see `receipts/2026-11-14-sketch-part-datums/`. Ordered model replay now solves dependent sketches after their source datums, and the session can adopt the result for a current generation; see `receipts/2026-11-14-ordered-sketch-recompute/`. Part owners now run ordered replay before publishing, and adoption joins open transactions without adding an undo step for a derived-only solve; see `receipts/2026-11-14-group-sketch-recompute/`. The desktop edit watch now resolves face-backed placements through ordered part replay and retires on source failure; see `receipts/2026-11-14-face-sketch-continuity/`. Desktop creation, inspection, editing, and mouse-driven rectangle placement now work on a face-backed work plane. Native `new-sketch-07` and `new-sketch-08` passed on the previous source pin; their receipts are stale after the source sync; see `receipts/2026-11-14-direct-face-d46c554381029cf4/` and `receipts/2026-11-14-face-support-work-plane-44dfe89fda9e10be/`. Direct-face preselection and wider sketch-tool coverage remain open. Stable face placement, replay, and native selection are tracked in #3884. See `receipts/2026-11-14-sketch-face-diagnostic/`.
- A9: all 43 viewport orientation, pan, zoom, and navigation-wheel declarations are assigned to R7 and remain inventoried (#3881). The four files match the live source as of 2026-11-14.
- A9: all 18 ribbon key-tip and nine viewport-orbit declarations are assigned to R7 (#3856, #3872, #3873). The reverse-rotation outline and one key-tip scenario have current native receipts; the rest are inventoried, not verified. Both source files match the live source as of 2026-11-14.
- R5 Coincident: three feature files have 36 reviewed milestone assignments (#3885). Corrected Coincident 07 passed all twelve steps on the current pin without the stashed preference integration; see `receipts/2026-11-14-corrected-coincident-07/`. A stronger native test found that two free points meet at their midpoint rather than the first selected point; the strengthened test and local integration are preserved in stash `3312db4726d16de7f129fa0df2d51b3702974add`. The Go solver also chooses the midpoint without an edit preference, and a Rust model regression checks that baseline (63d5e3241). Rank-aware linear and local nonlinear preference solves are tested, including dependent and conflicting targets (149f49eff); point-preference model solving and atomic constraint transactions are tested (2a0118ab9, 6464e7b97). API candidate a419c6e is preserved on local `Oblikovati.API` branch `feat/sketch-authoring-preference` and passed its full gate; session transaction integration is tested (4ae3da4e7); a local router and native candidate passed Coincident-07 with a temporary API override, including a recheck on 700db6032 after the solver change. The integration is stashed and not current acceptance; see `receipts/2026-11-14-preference-recheck/`. See `receipts/2026-11-14-local-api-integration/`. The full API duplication scan measured 6.2% against the 3% push limit (#301 in Oblikovati.API); a reduced scan excludes bindings, tests, and scripts and reports 0.58%, but does not satisfy the full rule. The candidate remains local; see `receipts/2026-11-14-api-dup-scope/`. See `receipts/2026-11-14-api-quality/`. See `receipts/2026-11-14-preferred-constraint/`. See `receipts/2026-11-14-coincident-diagnostic/` and `receipts/2026-11-14-coincident-go-oracle/`.
- A9: all 27 document-opening scenarios are assigned to R4, R6, or R7 (#3889). The existing-part activation scenario passed with document-window identity checks; its receipt is stale after the desktop work interface change. See `receipts/2026-11-14-open-existing-part/`.
- A9: all 23 document-saving scenarios are assigned to R4 or R7 (#3888). First-save cancellation and empty-part save/reopen passed their native scenarios; see `receipts/2026-11-14-empty-part-reopen/`. Sketch save/reopen and save-on-close also passed; see `receipts/2026-11-14-save-sketch-reopen/` and `receipts/2026-11-14-save-on-close/`. The sketch capture shows the rectangle, and the current automated check requires all four edges; see `receipts/2026-11-14-save-sketch-edges/`. Other receipts from before this test change are stale. The save-path request now clears when the GUI opens Save As, so cancellation returns to Ready; see `receipts/2026-11-14-save-cancel-notice/`.
- A9: all nine Point scenarios are assigned to R4, R5, or R7 (#3887). The full-feature diagnostic found an unbound first step in each case; see `receipts/2026-11-14-point-diagnostic/`. Later checks after those gaps are tainted. Point 04 and 05 now pass on the current inputs; see `receipts/2026-11-14-point-04/` and `receipts/2026-11-14-point-05/`. Point 03 passed before the latest test change and its receipt is stale.
- A9: all 39 linear-dimension and dimension-expression scenarios are assigned to R5. They remain inventoried and unverified (#3855, #3823).
- A9: all ten two-point rectangle scenarios have reviewed milestone assignments. The measurement and dimension-edit cases belong to R5; the remaining input cases belong to R4 (#3855, #3857).
- R4 extrusion: The `extrusion-01` receipt is current. The full BDD cube journey remains unverified (#3870).
- A9: all 30 extrusion scenarios have reviewed R4, R5, or R7 assignments. The extrusion-01 receipt is current; the others remain inventoried until acceptance passes (#3857, #3870).
- R2 built-in parameter persistence: schema 3 retains the model-parameter guard and rejects invalid or older-schema flags (#3883).
- R2 save evidence: 395 persistence tests and nine session save tests passed on Linux; see `receipts/2026-11-14-r2-save/` (#3592). This does not establish cross-platform replacement or the full R2 exit. The full workspace gate remains open.
- Kernel/profile mapping: Go drops a small square that Martensite admits, so one Martensite region alone does not verify a Go profile index. A translated square also fails Martensite area certification at the current pin and local main `4b76a823`. Diagnostic sources and results are in `receipts/2026-11-14-profile-parity-diagnostic/`; Martensite #446 is marked high priority.
- R3 legacy replay gap: nonempty Go sketch and feature records are refused by the package restore path; the existing cube replay uses a Rust recipe. A Go-generated constrained cube and externally captured face keys are in `receipts/2026-11-14-go-cube-fixture/`. Go saves the evaluated extrusion distance, so after reopen a size edit changes only the sketch dimensions. An isolated, strict candidate converter now covers point/line sketches, fixed, coincident, horizontal and vertical relations, and distance dimensions; it does not migrate or open a document. See `receipts/2026-11-14-legacy-sketch-candidate/`. A staged new-body distance extrusion uses an injected, Go-compatible profile resolver; the Go cube is checked through a fixture-only resolver and Martensite. See `receipts/2026-11-14-legacy-part-candidate/`. Go side-face witnesses for the original and a reordered fixture resolve to source-named Rust faces through edit, undo/redo, and Rust recipe replay; see `receipts/2026-11-14-go-face-lineage/`. A separate Go fixture now stores a side-face reference in an offset work plane and verifies it after Go edit/save/reopen; Rust explicitly refuses that work feature without changing the package. See `receipts/2026-11-14-go-dependent-face/`. The named prism adapter can now query a uniquely oriented planar face frame through edit, undo/redo and recipe reopen; its in-plane axes differ from Go and need a coordinate mapping before legacy adoption. See `receipts/2026-11-14-face-frame/`. The ordered part program now replays Rust semantic face-based datum planes with producer dependencies, versioned recipes and local failure; the Martensite replay and a separate session invalidation test pass. See `receipts/2026-11-14-face-replay/`. Desktop and headless replay owners resolve named planar faces against a current part result after parameter edits. The desktop command test confirms the public offset command now binds a current planar face key and previews replay before writing. Desktop and headless API tests cover edit, undo/redo, save/reopen, rejected names and transaction abort; see `receipts/2026-11-14-face-offset-api/`. File-host datum queries use current part replay. Immutable queries without a replay owner explicitly refuse face-dependent programs. Native selection and a sketch hosted on the plane remain open (#3884). Other face-based constructors and Go key migration remain open. Production profile mapping, package migration, other feature and sketch content, and Go dependent-face replay remain open (#3882).
- R3 cube evidence: the headless certified replay passed on `77f03cf61`; see `receipts/2026-11-14-r3-cube-check/`. The native GUI-loop cube test passed on `be2d5caf9`; see `receipts/2026-11-14-keyboard-cube-check/`. A foreground X11 mouse/keyboard journey on `0e0500c33` creates, edits, saves and reopens the cube, then resolves a saved face key through MCPBridge. See `receipts/2026-11-14-foreground-cube/`. The full R3 exit checks and gate remain open (#3857).
- MCPBridge measures a saved face key after cube replay; the 20 mm cube face reports 400 mm². See `receipts/2026-11-14-face-area-live/`. Other `analysis.measure` types return explicit declines and remain open (#3876).
- A7: the scenario bundle, file manifest, and verifier are tracked in `test-utilities/benchmarks/scenario-snapshots/` and `tools/bdd/`. See issue #3864.
- A6: diagnostic runner reports all steps and taints results after a failure. See issue #3863 and `receipts/2026-11-13-bdd-diagnostics/`. Fixture blockers require an explicit fixture signal.
- A4: the action queue remains manual. `python3 -B tools/bdd/behavior_cli.py queue` generates scenario work by milestone and feature; it does not infer dependency readiness.
- A5: applicable agent rules are tracked in `AGENTS.md` and its linked rule files. The branch and feature documentation policies match the port plan.
- A1: appearance and material resolution committed at `718a5d545`; Body Properties passed 6/6 scenarios and 69/69 steps on the saved source snapshot. See `receipts/2026-11-13-body-properties/receipt.json` and issue #3859.
- The stale green pixel oracle is replaced in the targeted native tests (#3867). The rotation test now checks that a move of the fixed profile is refused without changing the sketch, history generation, or displayed solid (#3868).
- The native GUI save/undo test used obsolete toolbar coordinates on clean base `e01e3a6e9` (#3874); see `receipts/2026-11-14-notice-baseline/`. It now targets rendered controls and passes through the camera assertions. A separate native test checks that the first toolbar navigation arrow stays out of the viewport.
- Earlier native evidence: see `receipts/2026-11-14-side-body/`, `receipts/2026-11-14-side-rectangle/`, `receipts/2026-11-14-side-extrusion/`, `receipts/2026-11-14-side-orbit/`, `receipts/2026-11-14-side-keytips/`, and the `side-01` through `side-05` receipt directories. The foreground keyboard and mouse history run is in `receipts/2026-11-14-sketch-history-live/`. The 195 live feature files match the saved snapshot as of 2026-11-14.
- On clean base `016ff28dc`, the typed-prism test expected `InvalidGeometry` for a short curve. The current kernel reports `InvalidInput` at admission with a `DegenerateCurve` cause; the adapter and native tests now check that cause, API reporting, and recovery (#3875). See `receipts/2026-11-14-profile-decline/`.
- A12: split the file host, assembly contract tests, GUI frame engine, and browser snapshot validation (#3860). The side ribbon code is split from `chrome.rs`, and command state is split from `commands.rs`. The ribbon catalog, browser outline, relation marks, drawing tests and persistence load tests are split; the desktop sketch work interface and implementation are split. Drawing and assembly opening tests are split; nine drawing tests and five assembly tests pass. The next `docs-lint` failure is `crates/head/tests/application/ribbon-customization/native_tests.rs`. See `receipts/2026-11-14-testsplit-checks/`.
- GUI unit tests pass serially (371/371); parallel execution can fail on simultaneous native context ownership (#3878).
- Per-face appearance presentation is not established by the body-color work. Verify it separately.
- Do not mark R0–R3 closed without their exit checks and receipts.

- R7 project dialog: selecting an inactive project keeps the original project active; all ten pinned scenario steps passed on the merged branch. See `receipts/2026-11-14-project-inactive-recheck/`, `receipts/2026-11-14-project-tree-b45b5713/`, `receipts/2026-11-14-project-tree-ffc9cb9e/`, and issue #3895.

- R7 project wizard: navigation through type, file and libraries pages, Back, and Cancel passed all 14 pinned steps on the merged branch. See `receipts/2026-11-14-wizard-check-5e1de952/` and issue #3895.

- R7 project folder browser: selecting Documents updates the workspace and derived project path; cancel keeps the previous paths. Both pinned scenarios passed on the merged branch; see `receipts/2026-11-14-folder-check-729a2f28/` and `receipts/2026-11-14-folder-check-d4a00f9d/` (#3895).

- R7 project-dialog scenario file: all eight declarations passed individually on the merged branch. Receipts are in `receipts/2026-11-14-project-final-*/`; the cancellation test also checked the visible provisional row and both empty library lists. This does not verify creating a project file (#3895).

- R4 document creation: five pinned scenarios passed individually on the native window; four create a part and the retained-template scenario creates an assembly. See `receipts/2026-11-14-create-r4-*/` (#3899). A feature-wide diagnostic reported 27 passes and 12 unbound scenarios; it is not acceptance evidence. See `receipts/2026-11-14-creation-diagnostic/summary.json`.

- R7 template chooser: secondary selection of both inch part templates, retaining selection after a click on empty gallery space, and Escape passed all 14 native steps. See `receipts/2026-11-14-template-secondary/` (#3899).

- R7 template chooser: keyboard highlight and mouse activation of the English folder retained the metric template details; all 11 native steps passed. The gallery assertion now inspects the gallery rather than the details pane. See `receipts/2026-11-14-template-folder-keyboard/` (#3899).

- R4 viewport picking: the host adapter now restricts intersections to the forward ray (#3925). The inside-solid and behind-origin adapter regression, host regression, native camera-ray test, and all 22 prism tests passed. Competing-category priority remains open (#3890).

- R3 cube journey: the native keyboard and pointer integration test passed after the forward-ray fix, including parameter edit, undo/redo, save/reopen, and named-face resolution. See `receipts/2026-11-14-forward-ray-cube/README.md`; this is not a BDD scenario receipt (#3870).

- Live source watch, 2026-11-14: `arc-center-point` scenarios 08 and 09 now identify the recorded pointer coordinates and 1700 × 1012 window; the current pin includes that correction. `user-coordinate-system` adds scenario 05 for canceling Redefine Feature (#3917). The current pin includes all 2,720 declarations; the new scenario is inventoried, not verified.
