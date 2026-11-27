# Historical BDD work ledger, part 2

This is an archived historical record, not current acceptance evidence. Old status labels and temporary report paths below are retained only for provenance. Use the [current ledger](../bdd-work-ledger.md) for current status.

## Evidence records

### Ribbon appearance baseline, 2026-11-13

- Source: `features/application/ribbon-appearance/ribbon-appearance.feature`,
  SHA-256 `846aff6433de2b69e222aa0580140c53adb7fdbe3aeba04ccab2a69ddc8c9438`.
  Application: `0ab980ca7`; Martensite uses the workspace lockfile.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime, `XMODIFIERS` unset.
  The complete native feature run passed seven of 18 scenarios. Ten stopped
  at undefined steps and one failed because the Tools ribbon lacks Team Web.
  A stopped scenario gives no evidence for its later steps.
- Run: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with `OBK_BDD_ROOT` pointing to the live source and
  `OBK_BDD_FEATURE=ribbon-appearance`. Reports are in
  `/tmp/obk-appearance-baseline-20261113`, a local, non-durable location.
  The appearance modes remain unavailable; the subsequent icon change alone
  does not establish their behavior.
- After the baseline, a native window test checked that the work and pattern
  controls remain unavailable and captured the model ribbon at
  `/tmp/obk-model-ribbon-icons-20261113/model-ribbon-work-and-pattern-icons.png`.
  The capture was visually inspected. The icon rasterization and draw-batch
  tests, 43 ribbon interaction tests, and the native cube journey passed.
- A subsequent native regression found that text-only command rows remained
  30 pixels apart at twice the icon scale. The shared command grid now scales
  text-only and icon command positions and button sizes together. The native
  scale regression, drawing Base View test, 43 ribbon interaction tests, and
  the 15-step panel-title and expansion scenario passed. The scaled ribbon
  capture at `/tmp/obk-scaled-command-grid-20261113/scaled-text-command-grid.png`
  was inspected.
  Focused scenario reports are in `/tmp/obk-ribbon-scaled-baseline-20261113`.
  This does not implement the appearance modes.
- A later native regression showed that switching from an open Appearance
  submenu to Show Panels needed two clicks. The first click dismissed the
  child popup without activating the parent row. The parent now handles that
  click while a child submenu is open. A native test passes in both directions
  and verifies that clicking the viewport closes the parent menu. The Escape
  and Reset Ribbon cancellation scenarios also pass. With the current source,
  the focused submenu scenario now passes its first seven steps and stops at
  the missing Team Web panel. Its report is in
  `/tmp/obk-submenu-exact-20261113`; it is not durable shared evidence.

### Viewport Orbit, 2026-11-13

- Source: `features/application/viewport-orbit/viewport-orbit.feature`,
  SHA-256 `d0e82e1671dc51da3c8eafa7d25101dc66713552639adf9dec041fac6d05eeb6`,
  unchanged before and after the run. Application: `a0458d3d4` plus updated
  scenario bindings; Martensite uses the workspace lockfile.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime, `XMODIFIERS` unset.
  All ten expanded cases and 94 steps passed, with no skipped steps. A later
  focused run of `Open and exit the free-rotation mode` passed all 13 steps
  after its drag step gained an assertion that the camera actually rotates.
  A separate native test confirmed that file-dialog Cancel still works.
- Run: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with `OBK_BDD_ROOT` pointing to the live source and
  `OBK_BDD_FEATURE=viewport-orbit`. Full-run reports are in
  `/tmp/obk-orbit-current-20261113`; focused reports are in
  `/tmp/obk-orbit-drag-20261113`. These local reports are not durable shared
  artifacts. This evidence covers this feature only.

### Viewport Zoom, 2026-11-13

- Source: `features/application/viewport-zoom/viewport-zoom.feature`, SHA-256
  `cb07444a13797a6d98fac5b8f26463343a1cbff5ed4781c359290bd6e4856e9c`,
  unchanged before and after the run. Application: `28ff1ad31` plus updated
  scenario bindings and fixture measurement; Martensite uses the lockfile.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime, `XMODIFIERS` unset.
  All seven scenarios and 74 steps passed with no skipped steps. A separate
  five-test fixture visibility suite passed. Pan was rerun and its four
  scenarios and 32 steps also passed after the shared measurement change.
- A green ribbon icon had contaminated the fixture's image bounds and caused
  the face-selection tests to click outside the block. The bounds calculation
  now excludes GUI windows; a regression test includes a green chrome pixel.
- Run: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with `OBK_BDD_ROOT` pointing to the live source and
  `OBK_BDD_FEATURE=viewport-zoom`. Reports are in
  `/tmp/obk-zoom-occlusion-final-20261113`; Pan reports are in
  `/tmp/obk-pan-after-bounds-20261113`. These local reports are not durable
  shared artifacts. These results cover these two features only.

### Viewport Pan, 2026-11-13

- Source: `features/application/viewport-pan/viewport-pan.feature`, SHA-256
  `4303c8a64b9262841dba3b8b48a74e51581a6a0ef9d98798286d514c53155647`,
  unchanged before and after the run. Application: `af00f6ca7` plus updated
  scenario bindings; Martensite uses the workspace lockfile.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime, `XMODIFIERS` unset so
  the X server supplies its local input method. All four scenarios and 32
  steps passed, with no skipped steps. The cases use mouse and keyboard events.
- The separate native `native_pan_drag_moves_the_view_and_keeps_pan_active`
  test also passed. It drags the view, checks camera and target translation,
  orientation and scale, and exits Pan with Escape.
- Run: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with `OBK_BDD_ROOT` pointing to the live scenario source,
  `OBK_BDD_FEATURE=viewport-pan`, and
  `OBK_BDD_REPORTS=/tmp/obk-pan-local-xim-20261113`. The local reports are
  not durable shared artifacts. This result covers this feature only.

There is no current full-suite native report in this ledger. The most recent
default report seen during setup contains one passing application scenario;
it is a filtered run and does not establish an area total. Existing targeted
runs remain useful development evidence but need their source and application
revisions checked before they support a current pass claim.

### Two-window zoom, 2026-11-13

- Source: `features/application/document-tabs/document-tabs.feature`, SHA-256
  `30014c87d865f758818ee93193b5550de1fea7d56ad46c493cd504fd2908560d`.
  The inventory checkpoint and the post-run file agree.
- Application: commit `6b42b7132` plus the two-window scenario binding and
  fixture changes in this ledger's commit. Martensite uses the locked workspace
  dependency. The existing unrelated README and AGENTS files were also present.
- Environment: Linux, Xvfb at 1600 × 1000, and the configured Vulkan runtime.
- Filter: `document-tabs`, scenario name containing `Retain separate zoom
  levels`. The native runner passed one scenario and nine steps; no steps were
  skipped. The mouse draws a rectangle, scrolls the second view, and switches
  views through the ribbon and tab. The captured image shows the rectangle.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with `OBK_BDD_ROOT`, `OBK_BDD_REPORTS`, `OBK_BDD_FEATURE`, and
  `OBK_BDD_SCENARIO` set as described above. Local reports are in
  `/tmp/obk-bdd-two-window-20261113`; they are not durable shared artifacts.
- Scope: this is one filtered case, not a feature or area total. The other
  application and sketch scenarios still require current native evidence.

### Panel titles across tabs, 2026-11-13

- Source: `features/application/ribbon-appearance/ribbon-appearance.feature`,
  SHA-256 `5de53f4ed08ea45af82b8e9c2cf628f859c713d96118848bd32304fadcb0a4a0`.
  The source matched before and after the run.
- Application: commit `1d1643274` plus the scenario bindings in this ledger's
  commit, with the locked Martensite dependency. The unrelated README and
  AGENTS files were present.
- Environment: Linux, Xvfb at 1600 × 1000, configured Vulkan runtime.
- Filter: feature `ribbon-appearance`, scenario `Share the panel-title setting
  between two ribbon tabs`. One scenario and all 11 steps passed using the
  native window. The test reads the actual rendered panel titles and menu
  check mark and operates both tab context menus by mouse.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with the source, report directory and filters set in `OBK_BDD_ROOT`,
  `OBK_BDD_REPORTS`, `OBK_BDD_FEATURE` and `OBK_BDD_SCENARIO`. Local reports
  are in `/tmp/obk-bdd-panel-title-share-final-20261113` and are not durable
  shared artifacts. This filtered pass does not establish feature coverage.

### Ribbon docking feature baseline, 2026-11-13

- Source: `features/application/ribbon-docking/ribbon-docking.feature`,
  SHA-256 `7bacd18f1484678bc9013e406ea4ecb7db81485e05c03a05cb986ecbbe607635`,
  unchanged across the run. Application commit `139b82a42`; the unrelated
  README and AGENTS files were present.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime. The full feature ran
  with `OBK_BDD_FEATURE=ribbon-docking` and no scenario-name filter. The
  native runner reported 23 cases: six passed and 17 stopped at undefined
  steps. A stopped case does not establish whether later behavior works.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`.
  Local JSON, JUnit, and screenshots are in
  `/tmp/obk-docking-feature-20261113`; these are not durable shared artifacts.
- Follow-up: corrected Modify-title bindings passed the two matching context
  scenarios, and the floating title-menu Escape scenario passed all 14 steps.
  These focused results used the same source and pending test-binding edits;
  they do not change the count of the earlier full run. A full rerun with those
  bindings reported eight passes, 14 undefined steps, and one ambiguous step.
  Its local report is `/tmp/obk-docking-current-20261113`. The quoted selection
  ambiguity was then removed, and the affected title-bar scenario passed all
  seven steps in a focused run. Rerun to establish the new full-feature count.

### File menu keyboard access, 2026-11-13

- Source: `features/application/file-menu/file-menu.feature`, SHA-256
  `19b41182a9710f6a5debb2e73e74b38c32a2d955583afe2e380ba01c850fa10f`.
  Application: commit `8691d1e2c` plus the File shortcut and test changes in
  this ledger's commit; the unrelated README and AGENTS files were present.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime. Native tests verified
  Alt followed by F, a held Alt+F chord, the visible F key tip, Escape and the
  unchanged active part. Four File dialog tests and nine key-tip tests passed.
- The mouse-driven `Open the file menu while a part is active` scenario passed
  all nine steps. The local report is `/tmp/obk-file-menu-20261113`; it is not
  a durable shared artifact. This result does not cover the New submenu.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with the live source, report directory, `file-menu` feature, and exact
  scenario-name substring set via the `OBK_BDD_*` environment variables.

### File New submenu, 2026-11-13

- Source: `features/application/file-menu/file-menu.feature`, SHA-256
  `19b41182a9710f6a5debb2e73e74b38c32a2d955583afe2e380ba01c850fa10f`,
  unchanged across the run. Application: commit `e8f258409` plus the New
  submenu and test changes in this ledger's commit; the unrelated README and
  AGENTS files were present.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime. The focused `Open the
  New submenu from the File menu` scenario passed all nine steps. The report
  and screenshot are in `/tmp/obk-new-submenu-final-20261113`; this location is
  local and is not a durable shared artifact.
- The native tests also selected a Part template and created a part, selected a
  Drawing template and verified its preview requirement, and switched from New
  to Save As. Weldment is disabled because there is no weldment template.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with the source, report directory, feature and scenario filters in the
  `OBK_BDD_*` environment variables. Other File menu cases need a full run.

### Complete File menu feature, 2026-11-13

- Source: `features/application/file-menu/file-menu.feature`, SHA-256
  `19b41182a9710f6a5debb2e73e74b38c32a2d955583afe2e380ba01c850fa10f`,
  unchanged across the run. Application commit `3eedfd14d`; unrelated README
  and AGENTS files were present.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime. All eight scenarios
  and 59 steps passed with `OBK_BDD_FEATURE=file-menu` and no scenario filter.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`.
  Local JSON, JUnit, and screenshots are in `/tmp/obk-file-area-20261113`;
  they are not durable shared artifacts. This result covers the File menu
  feature at that source and application revision, not the application area.

### Template name sorting and File menu rerun, 2026-11-13

- The native chooser test selects an inch template, shows the metric gallery,
  reverses the Name order twice and verifies that the inch details remain.
  The current catalog has only one standard and one sheet-metal part template;
  the full name-order BDD scenario requires additional template variants.
- After the sort change, all eight File menu scenarios and 59 steps passed
  again against source SHA-256
  `19b41182a9710f6a5debb2e73e74b38c32a2d955583afe2e380ba01c850fa10f`.
  The application is commit `3eedfd14d` plus the sort and test changes in
  this ledger's commit; unrelated README and AGENTS files were present.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime. The full feature used
  `OBK_BDD_FEATURE=file-menu` without a scenario filter. JSON, JUnit and
  screenshots are in `/tmp/obk-file-area-sort-20261113`, a local location.
  The command was `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked
  -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`.

### Template Details visibility, 2026-11-13

- Source: `features/application/document-creation/document-creation.feature`,
  SHA-256 `003a02efcb9c8f0629eeb8cabd4c52945533a1387d3918c417b7eac09a0d7522`,
  unchanged across the BDD run. Application: commit `0f6c0af70` plus the
  fixture and test changes in this ledger's commit; the unrelated README and
  AGENTS files were present.
- The native `Display templates as details` scenario passed all seven steps.
  The report and screenshot are in `/tmp/obk-doc-details-bdd-20261113`; that
  local directory is not a durable shared artifact.
- Focused native tests read complete Assembly and Presentation values in the
  Type column at the default 1440 × 900 window and after horizontal scrolling
  in an 800 × 600 window with 150% text scale. The latter screenshot is in
  `/tmp/obk-details-layout-final-20261113`. This is Linux/Xvfb evidence for
  issue #3858; it does not establish behavior on macOS or Windows.
- The name-sort BDD scenario still needs the distinct DIN/JIS template
  definitions. The Go reference does not provide those templates, so the
  application must not claim placeholder files as real templates.

### Template keyboard selection, 2026-11-13

- Source: `features/application/document-creation/document-creation.feature`,
  SHA-256 `003a02efcb9c8f0629eeb8cabd4c52945533a1387d3918c417b7eac09a0d7522`,
  unchanged across the run. Application: commit `753cf3b33` plus this ledger's
  navigation and test changes; unrelated README and AGENTS files were present.
- Native keyboard testing covers Left, Right, Home, End, group endpoints,
  selection details, and a nested Display menu. The Linux/Xvfb GUI suite passed
  347 tests; four File dialog tests and Clippy also passed.
- The current `Navigate backward and forward through template folders` BDD
  scenario passed all ten steps. JSON, JUnit, and screenshots are in
  `/tmp/obk-template-folder-navigation-20261113`, a local location. It ran
  with the live source, `OBK_BDD_FEATURE=document-creation` and the scenario
  name filter, using `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test
  --locked -p oblikovati-head --lib application_scenarios_use_the_native_window
  -- --nocapture`.
- The recorded adjacent-template and group-boundary scenarios still require
  the named DIN/JIS templates and a horizontal icon gallery. This test uses
  the existing metric templates and does not establish those full scenarios.

### Template icon gallery, 2026-11-13

- Source: `features/application/document-creation/document-creation.feature`,
  SHA-256 `003a02efcb9c8f0629eeb8cabd4c52945533a1387d3918c417b7eac09a0d7522`,
  unchanged across the runs. Application: commit `c58da4e42` plus the gallery
  changes in this ledger's commit; unrelated README and AGENTS files were present.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime. Three filtered native
  scenarios passed: `Change template icon size` (12 steps), `Select an assembly
  template in list view` (11 steps), and `Retain template sort order when
  reopening the chooser` (10 steps). The tests inspect actual icon draw quads.
- The native chooser regressions also covered double-click creation and
  keyboard scrolling at 800 × 600 with 150% text scale. The GUI library passed
  348 tests, the File dialog passed four, and Clippy passed.
- The File menu feature was rerun with source SHA-256
  `19b41182a9710f6a5debb2e73e74b38c32a2d955583afe2e380ba01c850fa10f`:
  all eight scenarios and 59 steps passed.
- The BDD runs used `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test
  --locked -p oblikovati-head --lib application_scenarios_use_the_native_window
  -- --nocapture` with the live source and feature/scenario filters. Local
  reports are under `/tmp/obk-final-icons-*` and
  `/tmp/obk-file-area-icons-20261113`; they are not durable shared artifacts.
- The DIN/JIS template variants remain unavailable. The related navigation
  scenarios are not claimed as passed.

### Secondary-button template selection, 2026-11-13

- Source: `features/application/document-creation/document-creation.feature`,
  SHA-256 `003a02efcb9c8f0629eeb8cabd4c52945533a1387d3918c417b7eac09a0d7522`.
  Application: commit `563929d48` plus the chooser and test changes in this
  ledger's commit. Unrelated README and AGENTS files were present.
- Three native tests passed on Linux with Xvfb at 1600 × 1000. They
  cover secondary-button selection in the icon, list and details views, empty
  space, popup state, visible selection and retention of a drawing preview.
  A screenshot is in `/tmp/obk-template-right-highlight-20261113`; this local
  location is not a durable shared artifact.
- The GUI library passed 349 tests, the File dialog passed four, and Clippy
  passed. The full File menu feature passed eight scenarios and 59 steps with
  its source hash unchanged, using Xvfb and the configured Lavapipe Vulkan
  driver. Reports are in `/tmp/obk-file-area-secondary-lvp-20261113`.
- The source scenario names `Standard (in)` and `Sheet Metal (in)`. The
  current catalog uses different names. The full BDD scenario is not claimed
  as passed; reconcile the catalog and bindings with the current source.

### Ribbon panel visibility baseline, 2026-11-13

- Source: `features/application/ribbon-panel-visibility/ribbon-panel-visibility.feature`,
  SHA-256 `3db03005baf02cddf7049fd563de2aeb9767e254c66458ceaa6fc365bbb316f0`,
  unchanged during the run. Application: commit `4d98e0a2c`; unrelated README and
  AGENTS files were present. The native run used Linux, Xvfb at 1600 × 1000,
  and Lavapipe. It executed the full feature with `OBK_BDD_FEATURE` set to
  `ribbon-panel-visibility` and no scenario filter.
- The runner reported 12 passed and 42 failed scenarios. Of the failures, 39
  stopped at an undefined step, two at the same ambiguous docking binding, and
  one at a failed Parameters icon assertion. Later steps in each stopped
  scenario were not tested. Local reports are in
  `/tmp/obk-panel-visibility-current-20261113`; they are not durable artifacts.
- The scenario file changed after this run to SHA-256
  `bdf21e3769aa826ac9106e0235328249209eccd14b471430d03442bc4fa71a79`.
  The earlier run is historical and must not be reported as the current result.
- Subsequent changes bind the corrected direct-selector wording, remove the
  duplicate docking binding, and locate the Parameters caption inside the
  ribbon instead of counting the open selector as a panel. The baseline above
  predates these changes and is not a pass count for the changed application.
- On the revised source, the native `Show and hide a ribbon panel` scenario
  passed 13 steps and `Retain panel visibility while switching tabs and parts`
  passed 10. The revised source hash was stable across both runs. Reports are
  in `/tmp/obk-panel-selector-revised-20261113` and
  `/tmp/obk-panel-retain-revised-20261113`. The side-docked scenario gets past
  the duplicate binding and stops at an undefined selector step. The GUI only
  draws the direct selector for the top placement, so the side placement
  needs implementation and native verification.

### Side ribbon panel selector, 2026-11-13

- Source: `features/application/ribbon-panel-visibility/ribbon-panel-visibility.feature`,
  SHA-256 `92f73c0f2a3167708cf0d04ffa2d6fde640b52a35c685637de13aadb2e224e24`,
  stable during the runs. Application: `ef91ac7d3` plus the selector and test
  changes in this ledger's commit. The unrelated README and AGENTS files were
  present. Linux, Xvfb at 1600 × 1000, Lavapipe Vulkan.
- The native `Toggle a panel from the left-docked ribbon selector` and right
  counterpart passed 22 steps together. `Hide and restore a panel on the
  side-docked ribbon` passed 12 steps, including group reflow. The runs used
  `OBK_BDD_FEATURE=ribbon-panel-visibility` with scenario filters and
  `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window --
  --nocapture`. Reports and screenshots are in `/tmp/obk-side-both-bdd-revised-20261113`
  and `/tmp/obk-side-reflow-revised-20261113`; these are local artifacts.
- The native mouse regression covers repeated toggles and dismissal on both
  sides. The GUI library passed 349 tests, the File dialog passed four, and
  GUI Clippy passed. Other panel-visibility scenarios remain unverified.

### Side selector and open groups, 2026-11-13

- Full feature source SHA-256
  `92f73c0f2a3167708cf0d04ffa2d6fde640b52a35c685637de13aadb2e224e24`
  remained stable during the run at application commit `6e80b9bde`. Linux,
  Xvfb at 1600 × 1000 and Lavapipe were used. The runner reported 15 passed
  and 39 failed scenarios: 38 stopped at an undefined step and one at a
  Parameters icon assertion. The local report is
  `/tmp/obk-panel-visibility-after-side-20261113`; it is not a durable artifact.
- Subsequent changes keep an open side group visible when its selector opens,
  close it on Escape, and close its commands when the group is hidden. Three
  native mouse regressions passed for these interactions and the selector on
  both sides. The GUI library passed 349 tests, the File dialog passed four,
  and GUI Clippy passed.
- On the same source hash, `Toggle the direct panel list without changing a
  checkbox` passed seven steps, and the left and right selector scenarios
  passed 22 steps together. The filtered native commands used
  `OBK_BDD_FEATURE=ribbon-panel-visibility` and `cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window --
  --nocapture` under Xvfb and Lavapipe. Local reports are in
  `/tmp/obk-side-group-dismiss-final-20261113` and
  `/tmp/obk-side-group-both-final-20261113`; they are not durable artifacts.
  The full feature baseline above predates these changes.

### Sketch ribbon visibility fixtures, 2026-11-13

- The current `Show and hide the Parameters panel during sketch editing`
  scenario requires the large icon inside an expanded panel. At the default
  1440-pixel window the panel collapses to a flyout; the previous icon check
  sampled that flyout button. The fixture now sets a 2400 × 900 window for the
  expanded Sketch panel scenarios and retains the icon assertion. The Go
  reference also collapses panels when the ribbon is too narrow.
- The native editor regression opens Parameters through the collapsed flyout
  at the default size and verifies parameter creation, undo and redo while the
  sketch remains active. The Layout native regression passes at the expanded
  width and verifies its disabled commands and Create Block dialog.
- The Parameters BDD scenario passed all 14 steps on source SHA-256
  `92f73c0f2a3167708cf0d04ffa2d6fde640b52a35c685637de13aadb2e224e24`,
  unchanged during the run. Application: `e86133af9` plus these test changes;
  unrelated README and AGENTS files were present. Linux, Xvfb at 1600 × 1000,
  Lavapipe, and a 2400 × 900 application window were used. The filtered run
  used `OBK_BDD_FEATURE=ribbon-panel-visibility`, `OBK_BDD_SCENARIO='Show and
  hide the Parameters panel during sketch editing'` and `cargo test --locked
  -p oblikovati-head --lib application_scenarios_use_the_native_window --
  --nocapture`. Local reports are in `/tmp/obk-sketch-param-bdd-final-20261113`;
  they are not durable artifacts.
- The Layout scenario now reaches the panel-order step after seven passing
  steps. Its required Sketch Insert panel is absent from the Rust ribbon, and
  its order step is not yet bound. Do not count this scenario as passed.

### Panel selector Home and End, 2026-11-13

- The selector now restores the focused panel after Home or End. Arrow keys,
  Space and Enter retain their existing navigation and activation behavior.
- Source: `features/application/ribbon-panel-visibility/ribbon-panel-visibility.feature`,
  SHA-256 `92f73c0f2a3167708cf0d04ffa2d6fde640b52a35c685637de13aadb2e224e24`.
  The filtered `Home and End leave the focused panel row unchanged` scenario
  passed all 11 steps on Linux, Xvfb at 1600 × 1000, and Lavapipe. Its native
  checks verify the focused widget identity as well as checkbox state. The
  report is in `/tmp/obk-selector-home-end-bdd-final-20261113`, a local path.
- The side selector also has native keyboard coverage for both docking sides.
  Seven focused GUI tests and that side test passed after the final change.
  A prior GUI run passed 350 library and four File dialog tests; the final
  change additionally clears stored focus when the selector is unavailable.
  GUI Clippy passed. The complete feature remains unverified after this change.

### Side ribbon additional commands, 2026-11-13

- Side-docked groups can open their additional commands. A pinned additional
  panel remains visible after its temporary group closes, and hiding the group
  removes the panel. Native mouse tests cover these interactions on both sides.
- The source changed after earlier evidence. On SHA-256
  `28915f5d5ef1f8360f7cb61481bdb48b6143c5a65c0167b25d4fdb16a30d2271`, stable
  during the run, `Dismiss the panel selector and expansion with Escape` passed
  all 10 steps. Application: `89dc46de1` plus this commit's changes; unrelated
  README and AGENTS files were present. Linux, Xvfb 1600 × 1000, Lavapipe.
  The filtered native command was `cargo test --locked -p oblikovati-head --lib
  application_scenarios_use_the_native_window -- --nocapture` with
  `OBK_BDD_FEATURE=ribbon-panel-visibility` and the scenario name filter.
  Local reports are in `/tmp/obk-side-additional-bdd-latest-20261113`; they are
  not durable artifacts. The GUI library passed 350 tests, the File dialog
  passed four, and GUI Clippy passed. Other scenarios require a full rerun on
  this source.
