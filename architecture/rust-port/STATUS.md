# Rust port status

Last checked: 2026-11-20. Recheck the branch, issues, and scenario source before
using this status; the live scenario directory can change.

## Start here

- Read `ACTION-PLAN.md`, `REVIEW-2026-11-17.md`, and the applicable `AGENTS.md`.
- Preserve the user's modified `README.md` and the untracked review file.
- Integrate completed work into `rust-work` and push; remove merged task branches,
  worktrees, and task-owned caches after checking them.
- Use the Go application as the reference and BDD scenarios as acceptance. The released host contract is [Oblikovati.API v0.166.0](https://github.com/Oblikovati/Oblikovati.API/releases/tag/v0.166.0) at `c8200b2`.
  Do not change Martensite; report kernel defects to its agent.

## Current evidence

- All 12 current [Viewport Navigation Wheel scenarios](receipts/2026-11-26-navigation-wheel-wayland-3881/README.md) passed on Wayland and X11 with native desktop input and the named Part.
- Viewport Orbit (#3872) passed all ten current cases and 94 steps on
  [Wayland and X11](receipts/2026-11-26-viewport-pan-orbit-wayland-3881/README.md),
  including both adjacent-pair examples. Direct drag and Escape tests passed on both backends.
- File Menu (#3894) passed all nine amended scenarios and 73 steps on Wayland and X11.
  The visual checks cover active controls, row alignment, and the overlay shadow.
  See the [current run](receipts/2026-11-26-file-menu-visual-contract-3894/README.md)
  and the [pinned run](receipts/2026-11-21-file-menu-r0-3894/README.md).
- Quick Access Toolbar (#3955) passed [all 25 cases and 268 steps on Wayland and X11](receipts/2026-11-26-quick-access-wayland-3955/README.md).
- Object Visibility (#3897) passed [all four published checklist scenarios and 29 steps](receipts/2026-11-26-object-visibility-work-planes-3897/README.md) on Wayland and X11. Native checks covered work-plane display and picking, origin axes, sketch dimensions, and temporary 2D sketch filtering.
- Shared Asset Browser (#3563) passed [five published scenarios and 32 steps](receipts/2026-11-26-asset-browser-shared-3563/README.md) on Wayland and X11. Material Browser passed [four scenarios and 30 steps](receipts/2026-11-26-material-browser-3563/README.md) on both desktops. Appearance Browser passed [six scenarios and 39 steps](receipts/2026-11-26-appearance-browser-3563/README.md) on both desktops. Generic Appearance Editor passed [four core scenarios and 42 steps](receipts/2026-11-26-appearance-editor-generic-3563/README.md) on both desktops. Appearance Information passed [its published scenario and native save/reopen check](receipts/2026-11-26-appearance-information-3563/README.md) on both desktops. Reflectivity passed [its published scenario and slider check](receipts/2026-11-26-appearance-reflectivity-3563/README.md) on both desktops. Transparency passed [its published scenario and custom refraction check](receipts/2026-11-26-appearance-transparency-3563/README.md) on both desktops. The four [optional Generic appearance cases](receipts/2026-11-26-optional-appearance-effects-3563/README.md) passed on both desktops. The [Generic Image menu and Checker editor](receipts/2026-11-26-appearance-checker-3563/README.md) passed two scenarios and 15 steps per desktop, with a separate native save check. The [preview scene and lighting controls](receipts/2026-11-26-appearance-preview-3563/README.md) passed two scenarios and 15 steps per desktop, with a separate Warm Light check. [Ceramic creation, finish, and disabled relief](receipts/2026-11-26-appearance-ceramic-3563/README.md) passed three scenarios and 24 steps per desktop, with a separate Part save check. [Ceramic image import and cancellation](receipts/2026-11-26-appearance-ceramic-maps-3563/README.md) passed the published Custom chooser scenario on both desktops, with a separate embedded-map save check. [Duplicate and Duplicate as Generic](receipts/2026-11-26-appearance-duplicate-3563/README.md) passed the two published scenarios on both desktops.
- Body Properties (#3954) passed all six current scenarios and 69 steps on Wayland and X11.
  The published Part verified calculated values, appearance changes, and dismissal without applying pending changes. See the [current runs](receipts/2026-11-26-body-properties-wayland-3954/README.md).
- Viewport Pan (#3881): all four pinned scenarios and 32 steps passed
  under Xvfb. Two X11 input tests also passed. The four declarations now
  have R0 records. The live feature requires a native Part fixture, and strict
  acceptance remains open under #3978. See
  `receipts/2026-11-20-viewport-pan-r0-3881/`.
- Viewport Orientation (#3961) passed all 27 amended scenarios and 291 steps
  on Wayland and X11. The HUD case covered Part and Assembly in both themes.
  See the [current receipt](receipts/2026-11-26-viewcube-hud-3961/README.md).
  The [20 pinned declarations](receipts/2026-11-20-viewport-orientation-r0-3961/) have R0 records.
  Strict build and runtime verification remains open under #3978.
- Workspace Layout (#3897) passed all [53 current cases and 523 steps on Wayland and X11](receipts/2026-11-26-workspace-layout-current-3897/README.md).
  The 52 declarations include two examples in one outline. The older
  foreground X11 check captured the View checklist and Mini-Toolbar with
  MCPBridge. Rule Log and Rule Automation are disabled under #4185.
  The [older pinned run](receipts/2026-11-20-workspace-layout-3897/) has 44 R0 baseline records. Strict verification
  still needs a current pin and bounded build and runtime inputs under Issue #3978.
- Document Tabs (#3900) passed all 41 amended scenarios and 352 steps on Wayland and X11, including the saved preview on a CIFS share.
  See the [current runs](receipts/2026-11-26-document-tabs-preview-3900/README.md).
  The pinned snapshot passed all 38 scenarios and 334 steps, with R0 baseline records in
  `receipts/2026-11-21-document-tabs-r0-3900/`.
  Strict verification remains blocked by #3978.
- Home Recent Documents (#3898) passed [all 37 amended scenarios on Wayland](receipts/2026-11-26-home-recent-resize-3898/README.md). The resize and both Home chrome scenarios also passed on X11.
- Document Opening (#3889) passed all [27 scenarios and 315 steps on Wayland and X11](receipts/2026-11-26-document-opening-current-fixtures-3889/README.md) with the current shared fixtures. Native pointer tests verified the Open thumbnail context menu on both desktops.
- Ribbon Display (#3856) passed [50 of 52 amended scenarios on Wayland and X11](receipts/2026-11-26-ribbon-command-geometry-3856/README.md), including split controls and the 170% text-size case. Two scenarios require Shell previews.
- Ribbon Panel Visibility (#3856) passed [53 of 55 cases and 626 steps in passing cases on Wayland](receipts/2026-11-26-side-model-panel-inventory-3856/README.md). The side-docked 3D Model inventory also passed on X11. Two cases still require Shell previews.
- Ribbon Panel Expansion (#3856) passed [59 of 67 cases and 668 steps](receipts/2026-11-25-ribbon-panel-expansion-current-3856/README.md) in the previous full run. [Highlight New](receipts/2026-11-25-highlight-new-3856/README.md) passed separately. Seven cases in the full run required Shell previews.
- Model Browser (#3896) passed [96 of 100 cases and 1,305 steps in passing cases](receipts/2026-11-26-model-browser-wayland-rerun-3896/README.md) in the full Wayland rerun. The four remaining `d1` setups require the extrusion taper contract in #4127 and Martensite #463.
  The [focused native Model State runs](implementation-model-browser.md) also passed 12 cases and 190 steps per backend on Wayland and X11.
  [Find and related dock controls](receipts/2026-11-26-wayland-browser-find-3896/README.md) passed 13 scenarios on Wayland. [Model-state arithmetic](receipts/2026-11-26-model-state-arithmetic-wayland-4119/README.md) passed on Wayland and X11.
- Ribbon Customization (#3856) passed 107 of 126 executions and 1,516 steps in the [full Wayland run](receipts/2026-11-26-ribbon-customization-current-3856/README.md). Subsequent checks passed for icon presentation and [Apply and panel removal](receipts/2026-11-26-ribbon-panel-retirement-3856/README.md) on Wayland and X11.
- Document Saving (#3888) passed 23 dependency-ready current scenarios and 260
  steps on Wayland. The generated Part requires Assembly Simplify (#3963).
  [Current run](receipts/2026-11-26-document-saving-wayland-3888/README.md)
  All 23 pinned scenarios and 218 steps passed in the
  [R0 run](receipts/2026-11-21-document-saving-r0-3888/README.md).
- Project Dialog (#3895) passed all eight current scenarios and 110 steps on
  Wayland and X11, including workspace folder selection and cancellation.
  See the [current runs](receipts/2026-11-25-project-workspace-folder-3895/README.md).
  Desktop startup opens native projects and the dialog wraps long paths.
  See `receipts/2026-11-20-native-project-dialog-3895/`. All eight scenarios and
  101 steps also passed on the pinned source under Xvfb. The eight declarations
  have R0 baseline records. See `receipts/2026-11-21-project-dialog-r0-3895/`.
  Strict verification still requires bounded build and runtime inputs under
  #3978.
- Docking System (#4133) passed all [14 amended scenarios and 140 steps on Wayland and X11](receipts/2026-11-26-docking-system-resize-4133/README.md),
  including a two-process desktop restart. The [floating Appearance Browser](receipts/2026-11-26-dock-floating-appearance-4133/README.md)
  and [per-document dock selection](receipts/2026-11-26-dock-document-tabs-4133/README.md) passed on Wayland and X11.
  The standalone [dock-menu test](receipts/2026-11-24-dock-menu-repository-fixture-4133/README.md)
  opens the bundled Part without requiring the separate docking scenario fixture.
- Selected sketch point identity (#4179): the shared placement recipe now
  retains explicitly picked point IDs and preserves existing geometry. Session
  tests cover preview, commit, dimension edit, and undo/redo. Midpoint Line
  now passes its selected point into this seam; the native origin test checks
  identity, zero degrees of freedom, and undo/redo. The 1,000-line
  selected-point preview median was 14.182 ms before rendering; GUI frame rate
  remains unverified. See `receipts/2026-11-20-picked-point-4179/` and
  `receipts/2026-11-20-midpoint-line-4178/`.
- Ribbon Appearance (#3957) passed all 18 current scenarios and 201 steps.
  See the [current Wayland and X11 runs](receipts/2026-11-26-ribbon-appearance-native-3957/README.md).
  The [earlier run](receipts/2026-11-20-ribbon-appearance-complete-3957/README.md)
  includes the pinned scenarios and the X11 Team Web launch check.
  All 18 pinned scenarios have R0 baseline records. Strict acceptance
  requires bounded build and runtime inputs under #3978.
- Ribbon Command Availability (#3956) passed all five current scenarios and
  39 steps on Wayland and X11 with the published Part, Assembly, and Drawing.
  See the [current runs](receipts/2026-11-26-ribbon-command-availability-wayland-3956/README.md).
  The [earlier run](receipts/2026-11-20-ribbon-command-availability-complete-3956/README.md)
  includes the pinned scenarios and X11 Text and Fillet checks.
- Ribbon Keyboard Navigation (#3856) passed all 18 current scenarios and 220 steps
  on [Wayland and X11 with the updated Part fixture](receipts/2026-11-26-ribbon-keyboard-current-3856/README.md).
- Ribbon Docking (#3860) passed [21 of 22 current scenarios and 223 steps on Wayland and X11](receipts/2026-11-26-ribbon-docking-native-3860/README.md). The Shell preview scenario remains blocked by Martensite #554.
  See the [current run](receipts/2026-11-25-ribbon-docking-current-3860/README.md).
- Sketch Text (#3907) opens the dockable Format Text editor from the Sketch
  ribbon. Point placement, preview, Cancel, OK, and creation undo and redo
  passed native input checks. Existing text now opens for editing from its
  glyph. Cancel restores the glyph. An accepted edit preserves the entity and
  anchor and uses one undo step. Pinned `text-08`, `text-09`, and `text-10`
  passed all 23 steps and have R0 baseline records. Pinned `text-02`,
  `text-04`, `text-07`, and `text-12` passed all 31 steps and also have R0
  records. They cover empty acceptance at 28 degrees of freedom and creation
  history with the same rendered text and position after Redo. The native
  editor offers installed fonts, height presets, and horizontal and vertical
  alignment. X11 checks cover visible preview changes, Cancel, history, and
  save and reopen. Invalid height keeps the last valid glyph and selects the
  rejected value after the warning closes. The selected-character formatting
  contract is available in [API v0.161.0](https://github.com/Oblikovati/Oblikovati.API/releases/tag/v0.161.0).
  The preset list, inline naming, and checked defaults passed pinned `text-24`.
  The native session preset applies to a fresh draft without committing geometry.
  Pinned `text-22` passed numbering and diameter-symbol insertion with DejaVu
  Sans, including a check of the curved outline and visual review of the canvas.
  The default font fallback is open under [Martensite #575](https://github.com/Oblikovati/Martensite/issues/575).
  Pinned `text-33` passed all eight steps. The native editor applied bullets
  to two selected paragraphs, replaced them with sequential numbers, and
  committed the Text at the selected point. Ready showed 42 dimensions
  needed. Pinned `text-31` saved and reopened a modified Text in the same
  application session and verified `*f` on reopen, `*g` after rename, and
  `g` after Save Current. The preset actions preserved the committed sketch.
  The full live Text run passed 11 of 38 declarations. The other 27
  lack step bindings. See `receipts/2026-11-20-sketch-text-preset-rename-3907/` for
  the prior full run and `receipts/2026-11-20-sketch-text-style-3907/` for the
  earlier formatting preview and save and reopen checks. Strict verification
  still requires bounded build and runtime inputs under #3978.
  Pinned `text-11` passed all eight steps after that full run. Bold thickened
  the selected text in the editor and on the canvas. Cancel preserved the
  original text and solver count. See
  `receipts/2026-11-20-sketch-text-bold-3907/`.
  Pinned `text-14` passed with an isolated Tahoma fixture. A native
  double-click after Redo showed both characters, the original font and
  height, and the Bold highlight. Cancel kept the outline and solver count.
  See `receipts/2026-11-21-sketch-text-tahoma-3907/`.
  Pinned `text-15` passed all eight steps with 33 dimensions needed.
  Italic preserved Bold. Undo restored the upright outline, and Redo
  restored the slanted outline. See
  `receipts/2026-11-20-sketch-text-italic-3907/`.
  Pinned `text-18` passed all five steps with an isolated Arial fixture.
  The font and height controls retained Bold and Italic, the 6.10 mm
  preset enlarged the visible outline, and Cancel kept the prior text.
  See `receipts/2026-11-21-sketch-text-arial-3907/`.
  API v0.162.0 provides the line-spacing contract and the desktop editor
  applies it to both editor rows and sketch outlines. Typed add, edit, and
  get tests cover history and native reopen. Pinned `text-21` checks
  Center, Middle, and Double spacing through the native editor. See
  `receipts/2026-11-21-sketch-text-spacing-0162/`.
  Pinned `text-28`, `text-30`, and `text-36` cover stacked fractions,
  named presets, and width recovery. The accepted blue fraction kept its
  original text ID and anchor. Applying its Arial preset made a second
  fraction with the same blue color, height, and width. The width warning
  selected zero for replacement and preserved the preview. See
  `receipts/2026-11-21-sketch-text-fractions-0163/`.
  Pinned `text-39` reopened a native Part in the same session and deleted
  preset `g` from the second fraction's editor. Cancel preserved both
  fractions and the unrelated label with 47 dimensions needed.
  See `receipts/2026-11-21-sketch-text-delete-preset-3907/`.
  Pinned `text-32` passed all ten steps with an isolated Tahoma fixture.
  Named `g` supplies the saved fraction. Last Used displays the clock and
  `*g` and restores the accepted fraction at a different width. Clearing
  the default leaves `g` available and opens empty Text with Tahoma,
  3.05 mm, a purple swatch, and 100 percent width. The native fallback run
  uses System default on a host without Tahoma.
  See `receipts/2026-11-21-sketch-text-restored-defaults-3907/`.
  Native Text placement opens Format Text when the mouse button is
  released. A drag keeps its initial insertion point. Escape while
  dragging cancels placement.
  See `receipts/2026-11-21-sketch-text-drag-3907/`.
  [API v0.165.0](https://github.com/Oblikovati/Oblikovati.API/releases/tag/v0.165.0)
  carries model-parameter and file-name fields. The native linked Text
  test creates a real sketch dimension, inserts fields at three and two
  decimals, edits the dimension with the mouse and keyboard, and checks
  Undo and Redo. Saving and reopening the part retains the field sources,
  resolves the saved file name, and permits another dimension edit.
  Pinned `text-17` passed all six steps. Editing `d0` to 12 refreshed the
  visible linked Text to `12.000`. Undo restored `10`, `10.000`, and the
  original sketch. See `receipts/2026-11-21-sketch-text-linked-dimension-3907/`.
  Pinned `text-34` and `text-35` passed all 20 steps on the same Part6
  setup. The filename field kept its source when switching to Finish and
  User. The model field used two decimal places. The viewport kept the
  earlier `10.000` and showed 47 dimensions needed at Ready. See
  `receipts/2026-11-21-sketch-text-filename-precision-3907/`.
  Pinned `text-16` passed all seven steps. It used a portable Part6
  with 36 dimensions needed and a 1440 × 1000 X11 window for the
  recorded drag. The linked `d0` showed `10.000` at Ready and the
  final solver count reached 38. See
  `receipts/2026-11-21-sketch-text-linked-drag-3907/`.
  Native `text-01` and `text-13` replays start at the recorded solver
  counts. Point placement adds one free anchor and reaches 212 and 30
  dimensions needed. Undo and Redo restore the original and accepted Bold
  states. The pinned source expects three more freedoms in both cases.
  See `scenario-defects/text-creation-degrees-of-freedom.md`.
  The native replay of `text-37` preserves the saved Arial fraction and
  47 dimensions needed after Cancel. Typing `6` yields a 6 mm preview.
  The scenario expects 6.10 mm before Cancel and awaits source correction.
  See `scenario-defects/text-37.md` and
  `receipts/2026-11-21-sketch-text-height-fraction-3907/`.
  Pinned `text-38` passed all nine steps with a generated native
  `Part6.opd`. After reopening Sketch2, the two fractions, numbered
  paragraphs, linked values, and 47 dimensions needed were retained.
  See `receipts/2026-11-21-sketch-text-expanded-reopen-3907/`.
  Format Text offers quarter-turn angles next to its numeric rotation field.
  A native RIGHT-view test selects 270 degrees and Center, checks that the
  preview is upright and horizontal, and confirms the committed angle.
  The pinned `text-26` scenario still needs the Text Box control and
  Background Fill context identified.
  Pinned `text-05` passed all six steps. A blank Tahoma preset supplied the
  configured default. After Cancel, the next placement was empty and showed
  Tahoma, 3.05 mm height, and 100 percent width.
  See `receipts/2026-11-21-sketch-text-cancel-defaults-3907/`.
  Pinned `text-06` passed all ten steps on a generated XY sketch with 210
  dimensions needed. The native preview showed `a` at (450, 650). Cancel
  removed its pixels and kept Text active. Escape kept the original sketch
  and the solver count.
  See `receipts/2026-11-21-sketch-text-cancel-preview-3907/`.
  Pinned `text-29` passed all six steps. Undo restored the blue diameter
  label. Redo restored the fraction and 35 dimensions needed. The linked
  `10.000` field retained its source. The native keyboard test also checked
  that the redone fraction matched its original glyph pixels.
  See `receipts/2026-11-21-sketch-text-fraction-history-3907/`.
  Native input left unselected `ab` unchanged and converted selected
  characters to `AB` in the editor and preview. The accepted edit kept
  the text ID and anchor. Pinned `text-25` also requires strike-through.
  Font decoration metrics are tracked in
  [Martensite #578](https://github.com/Oblikovati/Martensite/issues/578).
  The native palette test applied a blue swatch to selected text and found
  blue glyph pixels in both the live preview and accepted sketch. The YAML
  test retained color and width together. Pinned `text-19` still needs
  underline.
  The model uses the shared API to validate Unicode character-format ranges.
  Tests cover atomic rejection of out-of-range edits, undo and redo, and
  native YAML reopen. Mixed-font rendering remains open under
  [Martensite #576](https://github.com/Oblikovati/Martensite/issues/576).
- Midpoint Line (#4178): the scenario source changed and now requires an
  Inventor fixture for all six scenarios. Inventor interop is deferred. The
  Midpoint Line tool has native mouse and keyboard tests. A native YAML
  fixture is in `test-utilities/fixtures/empty-planar-sketch-right.opd` and
  has been supplied for scenario-owner review; the current steps still name
  the Inventor file. All six scenarios and 71 steps pass together in a temporary
  copy using the native fixture. Overlapping dimension labels were fixed in
  #4180. This copy does not verify the live source. See
  `receipts/2026-11-20-midpoint-full-diagnostic-4178/`.
- Sketch drag (#4171): partial motion is still refused. Conditional affine
  equation and optimum checks are merged, but they do not verify rounded sketch
  validity or prove interactive performance. The direct-drag worktree remains
  unmerged. The action-plan stop rule applies: do not continue isolated solver
  iterations without a passing scenario; return to a runnable feature.
- Constraint Settings source check: all eight scenarios now specify session
  preference baselines, but still require Inventor `.ipt` fixtures. Inventor
  interop is deferred; native fixture equivalents were requested in the feature
  `BLOCKERS.md`. The recorded 19.597 mm clicks also lack source coordinates.
- Constraint Settings (#4158): a full diagnostic on the uncommitted feature
  worktree passed five of eight scenarios with the feature and support hashes
  unchanged during the run. Three scenarios stopped at undefined setup steps;
  fixture questions are in the feature's `BLOCKERS.md`. This worktree result is
  not inventory evidence. The same result was reproduced after the checkbox
  diagnostic fix (#4175), with feature and support hashes unchanged. Point
  inference persistence (#4160), Relax Mode (#4164), and settings acknowledgement
  (#4172) remain in progress. Overlapping glyph selection (#4169) is merged.
  The merged branch has separate passing persistence, typed API, and X11 glyph
  checks in `receipts/2026-11-19-merged-glyph-0b3efd7/`.
- New Sketch (#4156): the full run at `60c48f25f` passed nine of ten
  scenarios. The accumulated document fixture remains unresolved (#4011) and
  requires circular sketch patterns (#3905). See the feature's `BLOCKERS.md`
  and the full-run receipt.
- Explode Block (#4153): the full run at `91e69a377` passed two of five
  scenarios. The other three need the portable fixtures requested in the
  feature's `BLOCKERS.md`. Native checks also cover canvas and browser
  explosion, color inheritance, selection of another instance, and analytic
  Measure before and after explosion. The latter fix is merged at `1d727f246`.
  The API contract is on `work/explode-block-310` at `0bf27c8`; its full API
  gate passed, but the contract is not released (#310, #311).
- Constraint inference scope (#4152): the full native run at `f7186111b`
  passed two of four scenarios. The recorded selection fixtures still need
  portable geometry; the scenario owner has been asked in `BLOCKERS.md`.
- Sketch preferences (#4148): the complete native run at `6fdb0dc6c` passed
  all three scenarios and 26 steps. Sketch edits are staged, Axes can be
  applied off and on, and HUD input controls retain their values on Cancel.
  Line scenario `line-30` now keeps the Background sketch, but its measurement
  fixture needs independent source coordinates (#4150).
- Sketch visibility (#4151): the complete native run at `e82b8006b` passed
  both scenarios and 17 steps. Isolation preserves earlier body visibility;
  a native regression also checks restoration when the sketch is finished.
- Model browser: the full native run at `0890f84e0` passed 97 of 100
  scenarios against feature SHA-256
  `603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
  Two failed test checks were corrected and passed focused reruns; the full
  feature has not been rerun after those fixes. The remaining extrusion-owned
  angle needs tapered extrusion (#4127, Martensite #463). The feature is not
  done. The scenario source is unversioned, so strict verified acceptance is
  still blocked by #3978.
- The full workspace-layout run at `c0060be36` passed 50 of 52 executed
  cases, including both outline examples. Its inventory counts 49 of 51
  declarations. The spacing and User Interface menu scenarios remain
  incomplete; their source questions are in the feature's `BLOCKERS.md`.
- Source edits have invalidated earlier runs, including runs previously
  reported as green. Counts are tied to source hashes and must be regenerated
  with `tools/bdd/feature_progress.py` before reporting them.
- Sketch constraint-glyph hover passed its complete one-scenario native run at
  `e33bac9dd`.
- Application Closing (#4030) passed all seven current scenarios and 51 steps
  on [Wayland and X11](receipts/2026-11-25-application-closing-current-4030/README.md).
  Strict build and runtime provenance remains open under #3978.
- Viewport Pan (#3881) passed all four current scenarios and 32 steps on
  [Wayland and X11](receipts/2026-11-26-viewport-pan-orbit-wayland-3881/README.md).
  Direct dragging and context-menu tests passed on both backends.
- Viewport Zoom (#3940) passed all seven current scenarios and 74 steps on Wayland and X11.
  The [current runs](receipts/2026-11-26-viewport-zoom-wayland-3940/README.md) include native wheel input on both backends.
  Other actions used window frame-loop input.
  All seven pinned scenarios also have R0 baseline records.
  See the [pinned run](receipts/2026-11-20-viewport-zoom-r0-3940/README.md).
- Viewport navigation wheel (#3881): all 12 scenarios and 143 steps passed
  with X11 mouse, Escape, fixture path entry and Enter, a native part, and
  unchanged source/support hashes. Compact wheel segment checks use rendered
  pixels. Full-wheel partitions, non-overlapping labels, and visible menu and
  close controls were checked in the native window. The source and build are
  unsealed, so this remains blocked from verified acceptance. See
  `receipts/2026-11-20-wheel-visual-3881/`.
  A later feature-scoped local snapshot pins the same feature, support files,
  and native fixture; the earlier run did not use that snapshot. See
  `receipts/2026-11-20-wheel-local-snapshot-3881/`.

- Main Window Frame (#4063) passed all 20 amended cases and 171 steps on
  [Wayland and X11](receipts/2026-11-26-main-window-frame-minimum-layout-4063/README.md).
  At 640 × 480 the side docks narrow, the active tab stays visible, the compact
  ribbon fits, and the floating Shell panel stays inside the viewport.
  The Wayland minimum drag used a private patched Weston test compositor.
- Viewport Entity Selection (#4055) passed all four current solid-face scenarios and 24 steps on [Wayland and X11](receipts/2026-11-26-viewport-entity-selection-current-4055/README.md).
  The native spatial fixture opens, but its 3D sketches are absent from the browser and viewport
  (#4092). See the [current run](receipts/2026-11-25-viewport-entity-selection-current-4055/README.md).
- Corliss 0.1.0 now reads point and line records from the Inventor fixture's
  two populated 3D sketches, with names and endpoint identities. This is a
  read-only prerequisite, not native document opening or viewport support;
  see `implementation-inventor-spatial.md` (#4092). A validated model now
  retains the recognized points and lines with shared endpoint identities.
  Corliss solid conversion is requested in [Martensite #545](https://github.com/Oblikovati/Martensite/issues/545).
- The ledger reports **0 verified of 2,760** declarations at local snapshot
  `b576445b98cffd686dcc05aa8c017de2a591c8c8`. This is not an upstream scenario
  commit. Run `python3 -B tools/bdd/behavior_cli.py report` for current counts.
- Review P0 F1–F4 are merged. Workspace all-target Clippy and `make docs-lint`
  passed. The post-merge docking diagnostic passed 8 scenarios and 65 steps;
  two capture-only cases were excluded. Input was mixed X11 and scripted events,
  and the unversioned source prevents verified acceptance. See
  `receipts/2026-11-17-p0-dock-lifecycle-4073/diagnostic.json`.
- All 104 checks in the four GUI groups from #4078 passed in the
  [current rerun](receipts/2026-11-26-gui-integration-targets-4078/README.md).
  Document tab checks now run as GUI unit tests.
- The head part-scene suite passed 134 of 135 tests; a modified body click
  selection test fails on both this work and clean parent `7ebc06bdb` (#4091).
- A focused X11 mouse and keyboard test creates a parameter-driven, fully
  constrained cube; edits its size; checks undo, redo, save/reopen, and named
  topology. Its [isometric capture](receipts/2026-11-18-os-cube-3857/cube.png)
  shows the resulting solid. This is a diagnostic, not a BDD scenario receipt;
  solve-precision qualification remains open (#3857, #3928).
- Fillet drawing, suppression-health, and extrusion-preview regressions pass
  using the complete named-solid evaluator (#4074). The evaluator requires
  explicit blend and named-face operations.
- Fillet and Chamfer creation reuse a current matching prefix. Tests count zero
  full body lookup, validation, and publication replays with a warm prefix;
  without it they count one of each. This does not measure frame rate (#4079).
- Feature-boundary checkpoints retain bodies, lineage, datums, and health.
  Session guards reject stale or foreign histories and restore exact names only
  for the accepted edit. Real-kernel tests compare named face areas with a full
  replay, including a Fillet consumed by a later Chamfer (#4079).
- Each document session owns up to four policy-scoped part results and their
  checkpoints. Hosts share a matching result; publishing a fifth policy retires
  the least recently published one. An edit invalidates all old generations,
  and closing the document drops all results (#4079).
- The document session runs and publishes normal full and ordered-sketch
  recomputes and creates feature checkpoints. The file workspace supplies the
  document identity and validates the policy; tests cover separate documents,
  stale generations, and checkpoint reuse (#4079).
- Checkpoint candidate replay and restoration now use a session-issued
  capability. The workspace rejects a foreign, wrong-policy, or stale prefix;
  tests also reject stale candidate histories (#4079).
- Certified Fillet and Chamfer edits use that capability for checkpoint replay.
  They require a current descendant and preserve injected validator refusals;
  tests cover foreign identities and a Fillet consumed by a Chamfer (#4079).
- Fillet and Chamfer creation now use a session-issued whole-part capability
  for append replay. Tests retain the replay counts and reject foreign or stale
  histories and validator refusals (#4079).
- Extrude creation also uses that capability, including the first empty-part
  replay. Tests cover policy and identity mismatches, an injected refusal, and
  named faces after a second extrusion (#4079).
- Captured append results are sealed to their document and policy. The file
  workspace restores and publishes them only for the accepted transition;
  tests reject foreign documents, wrong policies, and stale history (#4079).
- Feature-edit previews now replay through the document session, under its
  identity and policy. A private preview does not publish a result or change
  the document generation; the Fillet test compares full and checkpoint
  previews and checks cancellation and history (#4079).
- Extrude creation previews also use the session. A current append prefix is
  inspected under its generation guard without copying all bodies; tests compare
  named face areas with a full replay and cover policy isolation (#4079).
- Face-based datum previews use the same session preview service and current
  checkpoints. Tests check the offset geometry, unchanged document generation,
  and the face-plane command through history and reopening (#4079).
- Sketch inspection reads a retained part result when identity, resolution,
  solver options, and generation match. It recomputes for a stale or unmatched
  result; tests check face-sketch placement after a depth edit (#4079).
- Sketch continuity resolves all live watches in one batch per history refresh.
  Tests count one resolver call, preserve individual failures, and reject a
  malformed result count; the 60 fps target is not yet established (#4079).
- Fixed and origin-plane sketch continuity avoids a part replay when orientation
  has no feature-axis dependency. A mixed or feature-oriented batch still uses
  ordered replay; real-kernel tests count both paths (#4079).
- The scripted native Extrude preview test now opens Parameters from its
  Quick Access icon. Its former text lookup clicked the browser node; the test
  checks preview updates, stale requests, and cancellation (#4078).
- The native Fillet edit test now targets the radius input using its visible
  label. The previous click hit the edge count; the corrected scripted test
  checks preview, cancel, commit, undo, redo, and save/reopen (#4078).
- A cold checkpoint request for a part without associative sketch recompute
  publishes its result and boundaries from one replay instead of two. Ownership
  tests cover repeat requests and isolation between documents (#4079).
- A cold checkpoint request with a settled face sketch also publishes from one
  ordered replay. When sketch state changes, a second replay publishes the
  updated state; the native-owner test checks both paths (#4079).
- Certified Extrude creation publishes its validated document replay before
  display when a current prefix is available. Repeated previews reuse that
  prefix and cancel without an edit. Tests check two-extrusion names, policy
  isolation, and rejection by injected validators (#4079).
- Parameter Set and Update reuse a current document checkpoint for a proven
  clean prefix. A typed API test edits a Fillet parameter twice, checks named
  faces against a full replay, and exercises undo, redo, save, and reopen (#4079).
- End-of-Part edits use matching retained checkpoints for a proven clean prefix.
  Tests check rollback, restore, names, undo/redo, save/reopen, and full replay
  for transactions and mismatched policies (#4079).
- Feature deletion uses the same owner-checked replay path. Tests check later
  and first-feature deletion, replay counts, names, undo/redo, and reopen (#4079).
- Feature reorder reuses the unchanged prefix before the first moved feature.
  Tests cover replay counts, names, rejected moves, undo/redo, and reopen (#4079).
- Prism distance edits use the same checkpoint path. Tests compare every named
  face area with full replay and cover invalid inputs, history, and reopen (#4079).
- Drawing creation, live display, previews, and saved views obtain a current
  document-owned part, including when the injected owner uses another policy
  or is absent. Tests cover source edits, undo, retained results, and refusing
  to save an unhealthy source (#4079).
- On isolated Xvfb, 8 of 9 native drawing-opening tests passed. The remaining
  test could not select the clipped `layout` tab; it failed the same way on
  clean parent `01c398155` (related clipped-tab investigation #4082).
- The desktop replay owner creates checkpoints on demand for blend edit
  previews. Successive previews and accepted edits reuse matching checkpoints.
  Associative sketches use ordered recompute. Some preview and feature-specific
  orchestration still lives outside the session, so the complete evaluation
  service remains unfinished (#4079).
- A newly appended feature may add parameters without dirtying an unchanged
  prefix when no earlier feature reads them. Other parameter changes remain
  conservative (#4079).
- Standalone Projects passed [pointer docking, close, and saved-layout restart](receipts/2026-11-25-projects-docking-4073/README.md).
  [File Properties docks, closes, and reopens in the same tab group](receipts/2026-11-26-file-properties-dock-4073/README.md) on Wayland and X11.
  [Restored Projects accepts Done and Escape and returns to the selected template](receipts/2026-11-26-template-projects-dock-4073/README.md) on Wayland and X11.
- The 60 fps sketch target remains unproven. The recorded 1440 × 900 probe
  measured 20.738 ms median and 24.861 ms p95 against a 16.67 ms frame budget.
  See `receipts/2026-11-16-sketch-frame-baseline-4031/` and #4032.
- The full gate is not green. At `0c6943c4a` it stopped before workspace tests
  because the API revision differs from the released pin (#3965). The separate
  BDD tooling suite passed 342 tests; see
  `receipts/2026-11-20-full-gate-3965/`. Native build-provenance and receipt-reuse
  gaps remain #3978 and #3974. Do not count an excluded or unexecuted scenario
  as a pass.
- A Git package-selection diagnostic checks Cargo-selected files against locked
  blobs and detects source or vendor changes. Its host and build inputs remain
  unsealed; it cannot make a receipt reusable. See
  `receipts/2026-11-20-git-package-selection-3978/` (#3978).
- Git package checks reject a generated README path when the locked source has
  no supported README default, including a forged vendor checksum. Unsupported
  defaults and missing inherited defaults are refused rather than assumed (#3978).

## Work queue

Work dependency-ready application scenarios one at a time, then sketch and modeling.
1. Continue the dependency-ready application scenarios with Ribbon Customization.
   Midpoint Line (#4178) still names a deferred Inventor fixture. Revisit it when
   the scenario owner adopts the supplied native fixture.
   Constraint Settings (#4158) is paused with uncommitted work while its
   fixture questions and Relax Mode (#4164) remain open. Revisit `new-sketch`
   when fixture #4011 and circular patterns #3905 are available. Defer
   Inventor-file, assembly, and 3D-sketch scenarios.
2. Complete the evaluation owner and safe dirty-prefix replay in #4079, with
   generation, policy, invalidation, naming, and replay-count evidence.
3. Track the separate browser row movement diagnostic in #4077.
4. Complete the dock lifecycle checks still listed in #4073.
5. Continue ledger baseline and milestone exit work in #3865, #3861, and #3860.
6. Recheck scenario changes through the scheduled sync; the source may be edited
   while tests run. Capture source hashes before and after every diagnostic.

## Evidence and history

`feature-run-results.json` and `tools/bdd/feature_progress.py` report full-feature
diagnostics against current source hashes. The separate behavior ledger reports
strictly verified acceptance; unversioned scenario source prevents that proof.
`receipts/` contains recorded runs and their limitations. The prior status is
preserved in [STATUS-history-2026-11-17.md](STATUS-history-2026-11-17.md).
