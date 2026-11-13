# BDD work ledger

This ledger records active work and its evidence. Follow [the BDD workflow](bdd-workflow.md)
before changing a status. The source inventory is a discovery checkpoint, not
a pass count. The initial inventory records the current files; it does not
claim that each file has been individually triaged.

| Priority | Feature and scope | State | Evidence and next action |
| --- | --- | --- | --- |
| 1 | `features/application` | In progress | The native application runner exists. A complete, current area result has not been recorded. Run and retain the full area report; triage failures against the current source. |
| 1 | Native parameter-driven cube journey | In progress | The native journey creates a constrained rectangle and extrude, edits the size, uses undo/redo, saves and reopens, and resolves a face. Maintain the live test and rerun it after relevant changes. |
| 1 | `features/application/file-menu/file-menu.feature` | Verified | All eight scenarios passed on the current recorded source. Weldment remains unavailable. |
| 1 | `features/application/document-creation` | In progress | Name and size sorting, horizontal icon tiles, and keyboard and secondary-button selection work for available templates. The BDD scenarios still require additional named templates and native bindings. |
| 1 | `features/application/ribbon-customization/ribbon-customization.feature` | Needs rerun | Source changed since the inventory checkpoint. Review the corrections and rerun affected native scenarios. |
| 1 | `features/application/ribbon-appearance/ribbon-appearance.feature` | In progress | On the current recorded source, six of 18 scenarios pass; 11 stop at undefined steps and one fails on a missing Tools group. Reset Ribbon is implemented. Implement the standard Tools groups and Text Off, Small, Compact and Large across all commands and layouts. |
| 1 | `features/application/ribbon-docking/ribbon-docking.feature` | In progress | Source changed after the inventory checkpoint. A full native run at commit `139b82a42` found 6 passes and 17 scenarios stopped by undefined steps. With corrected bindings, a second run found 8 passes, 14 undefined steps, and one ambiguous step. Resolve the remaining steps and implement missing command-group behavior. |
| 1 | `features/application/ribbon-keyboard-navigation/ribbon-keyboard-navigation.feature` | Needs rerun | Source changed since the inventory checkpoint. Review the corrections and rerun affected native scenarios. |
| 1 | `features/application/ribbon-panel-visibility/ribbon-panel-visibility.feature` | Needs rerun | The source changed after the full run at `6e80b9bde`. Review the corrections and run the whole feature against a stable source; earlier counts are historical. |
| 1 | `features/application/ribbon-panel-visibility` side selector | In progress | The left and right selector and group reflow scenarios pass on the recorded source. Rerun the full feature and cover the remaining selector interactions. |
| 1 | `features/application/ribbon-panel-visibility` selector keyboard | In progress | Home and End retain the focused panel row; native verification and the current-source scenario are recorded below. Other keyboard scenarios need their named panel fixtures and a full feature rerun. |
| 1 | `features/application/ribbon-display/ribbon-display.feature` | In progress | Source changed. Revalidate all scenarios against the revised file, including appearance and modeling-preview cases. |
| 1 | `features/application/workspace-layout/workspace-layout.feature` | Needs rerun | Source changed since the inventory checkpoint. Review the correction and rerun affected native scenarios. |
| 2 | `features/sketch/pattern-rectangular/scenario.feature` | In progress | Rectangular patterns can be created and edited in the native UI, but directions are fixed to the sketch axes. Implement line-reference selection and associativity, then run the affected native scenarios. |
| 2 | Other `features/sketch` | In progress | The native sketch gate runs only selected folders. Inventory and run the remaining folders before reporting total coverage. |
| 3 | `features/solid-model` | In progress | Implementations exist, but this area has not been assessed against the current complete source. Assess each feature after the application and sketch priorities; no total execution result is recorded. |

## Evidence records

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

### Native cube journey and panel baseline, 2026-11-13

- The native mouse and keyboard journey passed after isolating its preferences
  and selecting Rectangle by its rendered label. It creates a part and an XY
  rectangle driven by `size = 10 mm`, checks coincident, horizontal, vertical
  and dimensional constraints and zero DOF, previews and creates an extrude,
  changes `size` to `20 mm`, uses undo and redo, saves and reopens the file,
  and checks solid properties and face keys. The test also resolves a face
  through the public host. It used application commit `ab9a6b426` plus this
  commit's test changes, Martensite revision `d051ff699982faba4ec2df3ee9a47e14f38dedbd`,
  Linux, Xvfb 1600 × 1000 and Lavapipe. The command was `xvfb-run -a -s
  '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib
  native_new_part_button_creates_and_selects_an_unsaved_document --
  --test-threads=1 --nocapture`. The local log is
  `/tmp/obk-native-cube-final-20261113.log`.
- A separate native regression at 150% text scale opens the collapsed Create
  panel and starts Rectangle with mouse input. Its local log is
  `/tmp/obk-scaled-sketch-final.log`; this does not repeat the full cube at
  that scale.
- A full panel-visibility run at `ab9a6b426` on source SHA-256
  `28915f5d5ef1f8360f7cb61481bdb48b6143c5a65c0167b25d4fdb16a30d2271`
  reported 19 of 55 scenarios passing. Of the 36 stops, 35 were undefined
  steps and one was a test binding that assumed the selector was side-docked.
  That binding now supports the top ribbon. The source changed after the run,
  so these counts are historical. The local report is
  `/tmp/obk-panel-visibility-ab9-20261113`.
- On source SHA-256
  `794416142395a581738cead4d629b3836bb29f59359c1663e8fe9cb95f46cd00`, the
  filtered selector-help scenario gets past opening the top-docked selector
  and stops at the next undefined step. The help behavior and remaining steps
  are not verified. The source changed again after this run, so the result is
  historical. Its local report is `/tmp/obk-selector-help-current-20261113`.

### Side context menu checkmark, 2026-11-13

- The `Hide and restore a panel from the left-docked ribbon context menu`
  scenario passed its first 12 steps on source SHA-256
  `770a7a833576e7a75b4c3c7f21d537b6f9c3c9c40d415f9a915b5646ecb7e485`,
  then failed `Modify is checked` because selecting the menu item closes the
  context menu. The source's functional notes say to reopen the menu to read
  the checkmark; the scenario has no such step after restoring the panel.
  Resolve this difference with the scenario source before claiming a pass.
  The source file changed after the run; this result is historical.
- A separate native mouse regression on both dock sides verifies that choosing
  Modify closes the menus, moves the remaining groups, restores their exact
  positions, and shows the checked state after reopening Show Panels. It ran on
  Linux, Xvfb 1600 × 1000 and Lavapipe. Its local log is
  `/tmp/obk-side-context-both-final.log`. Ribbon work is tracked in
  [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

### Hidden panel titles and expansion arrows, 2026-11-13

- Hiding panel titles retains the expansion arrows. Changing title visibility
  closes unpinned additional areas and retains pinned areas.
- The native `Hide panel titles while retaining access to additional commands`
  scenario passed all 15 steps on source SHA-256
  `5de53f4ed08ea45af82b8e9c2cf628f859c713d96118848bd32304fadcb0a4a0`,
  unchanged during the run. Application: `c66c28228` plus this commit's
  changes; unrelated README and AGENTS files were present. Linux, Xvfb at
  1600 × 1000 and Lavapipe were used. The filtered command was `cargo test
  --locked -p oblikovati-head --lib application_scenarios_use_the_native_window
  -- --nocapture` with `OBK_BDD_FEATURE=ribbon-appearance` and the scenario
  name filter. The local report is `/tmp/obk-hidden-title-arrow-bdd-final-20261113`.
- A native test also verifies that pinned additional commands survive title
  changes. The GUI library passed 350 tests, the File dialog passed four, and
  GUI Clippy passed. A native capture is in
  `/tmp/obk-hidden-title-arrow-capture-20261113`.
- The full feature run on the same stable source and application changes
  reported two of 18 scenarios passed. The other 16 stopped at undefined
  steps; their later steps were not tested. Reports are in
  `/tmp/obk-ribbon-appearance-full-20261113`. These paths are local artifacts.

### Appearance menu and tab changes, 2026-11-13

- Native BDD bindings now check that an inactive Sketch tab's context menu
  changes title visibility without selecting that tab, and that selecting
  another tab dismisses the appearance submenu. A sketch-editing variant also
  compares the sketch recipe before and after the tab changes.
- The full feature run on source SHA-256
  `5de53f4ed08ea45af82b8e9c2cf628f859c713d96118848bd32304fadcb0a4a0`,
  unchanged across the run, reported five of 18 scenarios passed. The other
  13 stopped at undefined steps. Application: `6efd3e9a9` plus this commit's
  test bindings; unrelated README and AGENTS files were present. Linux, Xvfb
  at 1600 × 1000 and Lavapipe were used. The unfiltered feature command was
  `cargo test --locked -p oblikovati-head --lib
  application_scenarios_use_the_native_window -- --nocapture` with
  `OBK_BDD_FEATURE=ribbon-appearance`. Reports are in
  `/tmp/obk-appearance-full-checked-20261113`, a local location.

### Switching side ribbon tabs with the selector open, 2026-11-13

- Clicking another side ribbon tab now closes the selector and activates that
  tab on the same click. This preserves the direct selector's panel choices.
- The native `Retain independent panel visibility across ribbon tabs` scenario
  passed all 13 steps on source SHA-256
  `3ed8b4aed2ec9ff418053a23399d1b4968043a3c6a18e57340e348209199888b`,
  unchanged during that run. Application: `857eb5996` plus this commit's
  changes; unrelated README and AGENTS files were present. Linux, Xvfb at
  1600 × 1000 and Lavapipe were used. The filtered run used
  `OBK_BDD_FEATURE=ribbon-panel-visibility` and `cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window --
  --nocapture`. The report is in `/tmp/obk-side-selector-tabs-fixed-20261113`,
  a local path. A separate native test covers both docking sides. The GUI
  library passed 350 tests, the File dialog passed four, and GUI Clippy passed.

### Reset Ribbon, 2026-11-13

- Reset Ribbon asks for confirmation and saves default customization before it
  changes the ribbon. A save failure keeps the applied ribbon and shows an
  error. Reset also restores display settings and the contextual tab.
- The native cancellation scenario passed all nine steps on appearance source
  SHA-256 `5de53f4ed08ea45af82b8e9c2cf628f859c713d96118848bd32304fadcb0a4a0`.
  The source stayed unchanged during the run. Native tests also cover reset,
  cancellation, save failure, dialog placement, and persistence after restart.
- The affirmative scenario reaches the restored Tools ribbon but fails because
  the standard Measure group is missing. It is not counted as passing.
- The Save As overwrite-cancellation scenario passed 12 steps on source
  SHA-256 `ba4bbe1607526458ea6f038a3e45fcadd53ac7f1eac825a0164f50a474faae2f`.
  Its source stayed unchanged during the run. The shared confirmation bindings
  were checked because Reset Ribbon uses them too.
- Application: `05e64216d` plus these changes. Linux, Xvfb at 1600 × 1000,
  and Lavapipe were used. The GUI library passed 351 tests, the file-dialog
  suite passed four, and GUI Clippy passed. Logs and captures under
  `/tmp/obk-reset-*` are local artifacts.

### Tools Measure command, 2026-11-13

- The Tools ribbon now has a Measure panel that calls the existing sketch
  inspection action. It is enabled during sketch editing and disabled in part
  mode, where measurement is not yet implemented. It does not satisfy the
  part-mode Measure scenarios.
- A native mouse test verified the disabled part-mode command, opened Measure
  while editing a sketch, and closed its panel with Done. Application:
  `c9e9e8c24` plus this change; Linux, Xvfb at 1600 × 1000, and Lavapipe.
  The local log is `/tmp/obk-tools-measure-native-20261113-2.log`.
- The remaining standard Tools panels and part-mode measurement are still
  required before the Reset Ribbon and ribbon display scenarios can pass.

### Scenario discovery checkpoint, 2026-11-13

- The live inventory reports changes to 16 application feature files since the
  committed baseline: document creation, opening, saving and tabs; file menu;
  Home recent documents; project dialog; quick access toolbar; ribbon
  appearance, customization, display, docking, keyboard navigation, panel
  expansion and panel visibility; and workspace layout. There are no added or
  removed files. Prior results for these files are historical until rerun.
- The baseline remains unchanged until each correction has been reviewed, as
  required by the BDD workflow. The local inventory output is
  `/tmp/obk-bdd-discovery-20261113.log`. The current source is not a Git
  checkout, so a prior source revision is not available there for comparison.

### Full ribbon appearance run, 2026-11-13

- Source: `features/application/ribbon-appearance/ribbon-appearance.feature`,
  SHA-256 `846aff6433de2b69e222aa0580140c53adb7fdbe3aeba04ccab2a69ddc8c9438`
  before and after the run. Application: `dc95f9e97`, with the unrelated
  README and AGENTS files present. Linux, Xvfb at 1600 × 1000 and Lavapipe.
- The full feature run passed six of 18 scenarios. Eleven stopped at an
  undefined step; the affirmative Reset Ribbon scenario failed when the
  Material and Appearance panel was absent. Its later steps were not tested.
  The 105 attempted steps were 93 passed and 12 failed.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window --
  --nocapture`, with `OBK_BDD_FEATURE=ribbon-appearance`, no scenario filter,
  and the live scenario root. Cucumber and JUnit reports are in
  `/tmp/obk-ribbon-appearance-full-dc95-20261113`, a local path.

### Dismiss the appearance submenu, 2026-11-13

- The current-source native scenario passed all eight steps: Escape first
  closes the appearance submenu, then closes its parent; the Tools ribbon
  remains in Normal appearance. The test opens the menu again to check the
  selected setting. This validates existing behavior and adds scenario
  bindings; it does not implement the missing appearance modes.
- Source: `features/application/ribbon-appearance/ribbon-appearance.feature`,
  SHA-256 `846aff6433de2b69e222aa0580140c53adb7fdbe3aeba04ccab2a69ddc8c9438`
  before and after the run. Application: `beb3028c0` plus these bindings, with
  unrelated README and AGENTS files present. Linux, Xvfb at 1600 × 1000 and
  Lavapipe. The filtered native runner report is in
  `/tmp/obk-appearance-escape-20261113`, a local path. The previous full
  feature count belongs to an earlier application revision.

### Tools Find command, 2026-11-13

- The Tools ribbon has a Find panel that opens the existing model-browser Find
  dialog. The dialog is drawn even when the browser is hidden, without
  changing browser visibility. Drawing documents do not offer this command.
- Native mouse tests passed for the Tools command with the browser hidden and
  for the existing browser-menu command. Both Reset Ribbon native tests and
  GUI Clippy also passed. Application: `6bbfb7558` plus this change; Linux,
  Xvfb at 1600 × 1000, and Lavapipe. Local logs are
  `/tmp/obk-tools-find-native-final.log` and `/tmp/obk-tools-find-reset.log`.
- The `Inspect the Find feature criteria` BDD run did not execute: the model
  browser feature file failed to parse on source SHA-256
  `42339c958b6c67613f2c93ef6732f6f22084742648327bff72cd94b146f0fdeb`,
  unchanged during the attempt. Its local report is
  `/tmp/obk-tools-find-browser-bdd-20261113`. Review the source and rerun it
  when the parse error is corrected.

### Current File menu scenarios, 2026-11-13

- All eight File menu scenarios passed, with 59 steps, on source SHA-256
  `fd33924caea15cf97ba75ad3600c7244cf17592b6bdf26bdc9882f7cae71fb91`.
  The source stayed unchanged during the run. Four revised interaction phrases
  were bound to the existing native mouse and keyboard steps. Weldment remains
  unavailable; the New submenu test checks that it is disabled.
- Application: `c0999b07f` plus these step bindings, with unrelated README
  and AGENTS files present. Linux, Xvfb at 1600 × 1000, and Lavapipe. Command:
  `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p
  oblikovati-head --lib application_scenarios_use_the_native_window --
  --nocapture`, with `OBK_BDD_FEATURE=file-menu`, no scenario filter, and the
  live source root. Reports are in `/tmp/obk-file-menu-full-final-20261113`,
  a local path.
- The latest inventory reports 23 changed application files against the
  unchanged discovery baseline, including model browser and five viewport
  features added to the earlier change list. The list is recorded in
  `/tmp/obk-bdd-discovery-filemenu.log`; each changed file needs review.

### Tools Clipboard panel, 2026-11-13

- The Tools ribbon now lists Clipboard in its normal panel order. Copy is
  visible but disabled because model copy is not implemented. The panel uses
  the shared Show Panels controls.
- A native mouse test verified the disabled Copy control, hiding and restoring
  Clipboard, and retaining the hidden state when switching tabs. Application:
  `bbb000d11` plus this change; Linux, Xvfb at 1600 × 1000, and Lavapipe.
  The local log is `/tmp/obk-tools-clipboard-native-20261113.log`.
- Copy's move-to-expanded-panel interaction remains unimplemented, as do model
  copy and the other missing Tools groups. No full BDD scenario is claimed.
