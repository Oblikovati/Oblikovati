# Historical BDD work ledger, part 3

This is an archived historical record, not current acceptance evidence. Old status labels and temporary report paths below are retained only for provenance. Use the [current ledger](../bdd-work-ledger.md) for current status.

## Evidence records

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

### Gherkin parse diagnostics, 2026-11-13

- On a failed native benchmark, the runner now reports the parser's underlying
  diagnostic, including the source path and line and column when provided.
  A unit test checks that the underlying read error is retained.
- The current model-browser source, SHA-256
  `42339c958b6c67613f2c93ef6732f6f22084742648327bff72cd94b146f0fdeb`,
  failed parsing at line 587 because the step keyword is lowercase `and`.
  The source stayed unchanged during the run. The feature needs correction
  before its scenarios can run. The local diagnostic report is
  `/tmp/obk-bdd-model-browser-parse-detail-20261113`.
- A valid File menu scenario passed all five steps through the same runner on
  SHA-256 `fd33924caea15cf97ba75ad3600c7244cf17592b6bdf26bdc9882f7cae71fb91`,
  unchanged during the run. Application: `f97653057` plus this change; Linux,
  Xvfb at 1600 × 1000 and Lavapipe. Its local report is
  `/tmp/obk-bdd-parser-valid-20261113`.

### Native cube journey rerun, 2026-11-13

- The native mouse and keyboard cube journey passed on application commit
  `34a4bbfe1155ffb28a338cdbdf9f233cf0887e82`, with unrelated README and
  AGENTS files present. It covers the constrained parameter-driven rectangle,
  extrusion, size edit, undo/redo, save/reopen and named-face resolution.
- Martensite revision: `d051ff699982faba4ec2df3ee9a47e14f38dedbd`. Linux,
  Xvfb at 1600 × 1000, and Lavapipe. The command was `xvfb-run -a -s
  '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib
  native_new_part_button_creates_and_selects_an_unsaved_document --
  --test-threads=1 --nocapture`. The local log is
  `/tmp/obk-native-cube-34a4-20261113.log`.

### Model-browser Find scenario, 2026-11-13

- Corrected two lowercase Gherkin `and` keywords at lines 587 and 930 in
  the live model-browser feature file. These syntax changes leave the step
  text unchanged. The source is outside the Rust repository and is not part
  of this commit. The phrase after the second keyword still needs a semantic
  review by the scenario author.
- The native `Inspect the Find feature criteria` scenario passed all ten
  steps after binding its revised interaction wording. The corrected source
  SHA-256 was `603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`
  before and after the run. The rest of the feature was not run.
- Application: `1d7a12698` plus these bindings; unrelated README and AGENTS
  files were present. Linux, Xvfb at 1600 × 1000 and Lavapipe. The filtered
  native report is `/tmp/obk-model-browser-find-native-20261113`, a local path.

### Built-in ribbon command relocation, 2026-11-13

- Copy can move between the Clipboard main panel and its expanded panel. Its
  unavailable state is preserved. The location is saved per document ribbon,
  included in customization import/export, and cleared by Reset Ribbon. A
  failed save retains the applied location and reports the error. Version 3
  settings retain version 1 and 2 import support. This supersedes the earlier
  Clipboard record that said relocation was unimplemented. Model copy itself
  remains unavailable.
- The native `Move an unavailable command to an expanded panel and back`
  scenario passed all 15 steps on source SHA-256
  `645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`,
  unchanged during the run. Application: `03f61c1e7` plus this change, with
  unrelated README and AGENTS files present. Linux, Xvfb at 1600 × 1000 and
  Lavapipe. The local report and expanded-panel capture are in
  `/tmp/obk-builtin-copy-bdd-guard-20261113`. Other scenarios were not run.
- Two native Clipboard tests passed, including a restart, and two Reset Ribbon
  tests passed. The GUI library passed 353 tests before the final command ID
  isolation; the added identity test and the BDD scenario passed afterward.
  The File dialog suite passed four tests, and GUI Clippy passed after the
  final change. Unit tests cover old settings versions, import/export, invalid
  IDs, save failure, reload, reset, and an unrelated pending customization.

### Ribbon settings import verification, 2026-11-13

- Corrected three lowercase Gherkin `and` keywords at lines 161, 178 and
  196 in the live ribbon-customization feature file. The remaining phrase
  `cancels and Apply` needs semantic review by the scenario author. The source
  is outside the Rust repository and is not part of this commit.
- The native `Stage an import of freshly exported customization settings`
  scenario passed all 13 steps on corrected source SHA-256
  `c338bc2ca7723cbc58a28c13b97869e61ad0697ed3290c67843149e1e608a73e`,
  unchanged during the run. The rest of the feature was not run.
- Native tests also passed for exporting and importing a user-command draft
  and for staging, applying, and restoring an exported built-in command
  location. Application: `1653a0368` plus these test bindings and checks;
  unrelated README and AGENTS files were present. Linux, Xvfb at 1600 × 1000
  and Lavapipe. The local report is `/tmp/obk-ribbon-import-v3-final-20261113`;
  the built-in import log is `/tmp/obk-builtin-import-native-20261113.log`.

### Working built-in command relocation, 2026-11-13

- The same saved location path now handles both unavailable Copy and the
  working sketch Measure command. The main and expanded controls dispatch the
  same Measure action and keep its sketch-only availability.
- Two native Measure tests passed, including moving Measure to the expanded
  panel, running it there, moving it back, and running it again. Three native
  Clipboard tests passed, including restart and import/export. The GUI ribbon
  customization tests passed 36 cases, the command identity test passed, and
  GUI Clippy passed. Application: `9eaeb2d1f` plus this change; unrelated
  README and AGENTS files were present. Linux, Xvfb at 1600 × 1000, Lavapipe.
- The native Copy relocation BDD scenario still passed all 15 steps on source
  SHA-256 `645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`,
  unchanged during the run. The local report is
  `/tmp/obk-builtin-working-copy-bdd-20261113`. Part-mode Measure remains
  unavailable; the full ribbon feature has not been rerun.

### Part face measurement, 2026-11-13

- Measure is available in a part and reports a selected solid face's analytic
  area in mm². The B-rep line pick and face report come from Martensite; the
  measurement does not use the display mesh. The dialog states that it accepts
  a solid face. Edge, distance, and angle measurements remain unimplemented.
- Model-face picking now works when the model browser is hidden. A native
  mouse test measured a 40 × 30 mm face as 1,200 mm², switched ribbon tabs,
  and cleared the selection. The captured image is the local file
  `target/benchmarks/application/part-measure-analytic-face.png`.
- The native `Switch ribbon tabs with a command panel open` scenario passed
  all 10 steps on source SHA-256
  `82542b170cbae71ede004fcb11be64a14bedbb10fdaf9b40df7b2eec462f636e`,
  unchanged during the run. Application: `b114cdba9` plus this change;
  unrelated README and AGENTS files were present. Linux, Xvfb at 1600 × 1000,
  and Lavapipe. The local report is `/tmp/obk-part-measure-bdd-scope-20261113`.
- Three native Measure tests and two existing face-selection tests passed.
  Unit checks cover invalid, duplicate, and non-finite face reports and stale
  model generations. The GUI suite passed 354 tests and the file-dialog suite
  passed four. The native cube journey and GUI/head Clippy passed after the
  viewport-picking change. The final prompt change was checked with the native
  Measure tests, the BDD scenario and head Clippy. Earlier records stating that
  part-mode Measure was unavailable are historical.

### Part material and appearance selectors, 2026-11-13

- The Tools ribbon offers part material and appearance selectors. They use
  document, project, and shipped catalog choices in that priority order and
  call the existing typed assignment API. Unknown assigned IDs remain visible
  in the selector. The Go-style Material Browser and asset editing controls
  are not yet implemented.
- Native tests selected material and appearance through the UI, used undo
  and redo, reset the ribbon, and saved and reopened the part. A second test
  verified that an appearance assignment refreshes a visible solid; its local
  capture is `/tmp/obk-part-assets-capture-final-20261113/part-appearance-solid.png`.
  A third test checked refusal of stale documents and unknown asset IDs.
- The focused Reset Ribbon BDD run passed 10 steps and failed its final Tools
  group check because Team Web is missing. Material and Appearance was visible.
  Source SHA-256 `846aff6433de2b69e222aa0580140c53adb7fdbe3aeba04ccab2a69ddc8c9438`
  was unchanged during the run. Local report:
  `/tmp/obk-part-assets-reset-bdd-final-20261113`.
- Application: `a7effffd3` plus this change; unrelated README and AGENTS files
  were present. Linux, Xvfb at 1600 × 1000, Lavapipe. GUI and head Clippy
  passed; the GUI library passed 356 tests and file dialogs passed four before
  the final selector refactor. The focused GUI and native asset tests and cube
  journey passed afterward. The full ribbon feature was not rerun.

### Material Browser quick access, 2026-11-13

- The quick access command opens a Material Browser for the active part. Its
  Materials and Appearances tabs select catalog choices and assign them through
  the existing part asset operation. Switching parts clears pending choices.
  Property editing, duplication, and the Physical tab are not implemented.
- The native `Open and close the Material Browser from quick access` scenario
  passed all 10 steps on source SHA-256
  `e7546a712fe3f42b2c6f9885e47292b69cb3ef0ebe3105a9ad9a26bbed40bbbd`,
  unchanged during the run. The other scenarios were not run. Application:
  `447ec4bc0` plus this change, with unrelated README and AGENTS files present.
  Linux, Xvfb at 1600 × 1000 and Lavapipe. Local report:
  `/tmp/obk-material-browser-final-bdd-20261113`.
- Two native tests passed for selecting a catalog material, assignment and
  undo, appearance assignment, and switching to a second part. The browser
  capture is `/tmp/obk-material-browser-final-native-20261113/material-browser-assigned.png`.
- Three existing native ribbon asset tests passed. The GUI library passed
  358 tests, the File dialog passed four, and GUI/head Clippy passed. The new
  toolbar command moved the Manage tab; the parameter journey's fixed test
  click was updated and its shortcut checks passed on both widget platforms.

### Quick access toolbar feature, 2026-11-13

- The scenario source changed its action wording. Native bindings now accept
  the current wording and retain the earlier wording. Added native checks for
  disabled Redo and the complete expected customization choices.
- The full feature run passed 11 of 12 scenarios and 122 of 123 executed steps.
  The remaining scenario fails because Projects is absent from the menu. Steps
  after that failure did not run; the other named choices still require work
  and verification. Source SHA-256:
  `e7546a712fe3f42b2c6f9885e47292b69cb3ef0ebe3105a9ad9a26bbed40bbbd`,
  unchanged during the run. Application: `d192eed2c` plus these bindings;
  unrelated README and AGENTS files were present. Linux, Xvfb at 1600 × 1000,
  Lavapipe. Local JSON and JUnit reports: `/tmp/obk-quick-access-current-20261113`.
- The inventory reports many source files changed since the discovery baseline.
  This is not a current suite pass count. In particular, File menu changed
  after its recorded full pass and requires a new full run.
