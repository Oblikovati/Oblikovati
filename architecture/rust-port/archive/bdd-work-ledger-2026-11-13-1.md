# Historical BDD work ledger, part 1

This is an archived historical record, not current acceptance evidence. Old status labels and temporary report paths below are retained only for provenance. Use the [current ledger](../bdd-work-ledger.md) for current status.

## Original ledger

This ledger records active work and its evidence. Follow [the BDD workflow](../bdd-workflow.md)
before changing a status. The source inventory is a discovery checkpoint, not
a pass count. The initial inventory records the current files; it does not
claim that each file has been individually triaged.

The live inventory on 2026-11-13 reports changes in 23 of 24 application
feature files against the discovery checkpoint, as well as changes in sketch
and solid-model files. Treat older feature results as historical until their
recorded source hash is compared with the live file and the required rerun is
complete. The checkpoint has not been advanced: the changes still need triage.

| Priority | Feature and scope | State | Evidence and next action |
| --- | --- | --- | --- |
| 1 | `features/application` | In progress | The native application runner exists. A complete, current area result has not been recorded. Run and retain the full area report; triage failures against the current source. |
| 1 | Native parameter-driven cube journey | In progress | The native journey creates a constrained rectangle and extrude, edits the size, uses undo/redo, saves and reopens, and resolves a face. Maintain the live test and rerun it after relevant changes. |
| 1 | `features/application/file-menu/file-menu.feature` | Verified | All eight scenarios passed on the recorded updated source. Weldment remains unavailable. |
| 1 | `features/application/document-creation` | In progress | The current-source run with gallery setup bindings passed 27 of 39 native scenarios; 12 stopped at undefined steps. Add the named template and populated-layout fixtures and remaining scenario bindings. See the current run below. |
| 1 | `features/application/document-opening` | In progress | The current-source native run passed 26 of 27 scenarios. Express assembly loading stopped at a missing component fixture; its later steps did not run. The saved-drawing scenario passes with its referenced part initially closed. |
| 1 | `features/application/document-saving` | Passing | The current-source native feature run passed 23 of 23 scenarios. The first-save preview is rendered from the part and stored in the saved file. See the save evidence below. |
| 1 | `features/application/model-browser` | In progress | After action wording updates, a full native run passed 29 of 100 scenarios. Of the other 71, 62 stopped at undefined steps and nine at failed checks. Focused fixes passed later; rerun the full feature before reporting a current total. |
| 1 | `features/application/quick-access-toolbar` | In progress | The current-source full native run passed 11 of 12 scenarios. Update, Select, Materials, Appearance Browser and Parameters work from Quick Access; the remaining customization-choices scenario stops at Refine Appearance. Later checks did not run. The CLI script host shares the part replay operation. Material Browser editing, duplication, and the Physical tab remain open. |
| 1 | `features/application/viewport-pan/viewport-pan.feature` | Verified | All four current scenarios passed in the native window after the step wording was updated; see the Pan evidence below. |
| 1 | `features/application/viewport-zoom/viewport-zoom.feature` | Verified | All seven current scenarios passed in the native window after the step wording and fixture bounds measurement were corrected; see the Zoom evidence below. |
| 1 | `features/application/viewport-orbit/viewport-orbit.feature` | Verified | All ten expanded cases passed in the native window after the step wording was updated; see the Orbit evidence below. |
| 1 | `features/application/ribbon-customization/ribbon-customization.feature` | Needs rerun | Source changed since the inventory checkpoint. Review the corrections and rerun affected native scenarios. |
| 1 | `features/application/ribbon-appearance/ribbon-appearance.feature` | In progress | On the recorded source, seven of 18 scenarios pass; ten stop at undefined steps and one fails on a missing Tools group. Reset Ribbon is implemented. Go icons now cover model work-plane, axis, point, pattern and Simplify controls. Implement the standard Tools groups and Text Off, Small, Compact and Large across all commands and layouts. |
| 1 | `features/application/ribbon-docking/ribbon-docking.feature` | In progress | Source changed after the inventory checkpoint. A full native run at commit `139b82a42` found 6 passes and 17 scenarios stopped by undefined steps. With corrected bindings, a second run found 8 passes, 14 undefined steps, and one ambiguous step. Resolve the remaining steps and implement missing command-group behavior. |
| 1 | `features/application/ribbon-keyboard-navigation/ribbon-keyboard-navigation.feature` | Needs rerun | Source changed since the inventory checkpoint. Review the corrections and rerun affected native scenarios. |
| 1 | `features/application/ribbon-panel-expansion/ribbon-panel-expansion.feature` | In progress | The unavailable Copy command can move between the Clipboard main and expanded panels and survives restart. Run the remaining scenarios on current source. |
| 1 | `features/application/ribbon-panel-visibility/ribbon-panel-visibility.feature` | Needs rerun | The source changed after the full run at `6e80b9bde`. Review the corrections and run the whole feature against a stable source; earlier counts are historical. |
| 1 | `features/application/ribbon-panel-visibility` side selector | In progress | The left and right selector and group reflow scenarios pass on the recorded source. Rerun the full feature and cover the remaining selector interactions. |
| 1 | `features/application/ribbon-panel-visibility` selector keyboard | In progress | Home and End retain the focused panel row; native verification and the current-source scenario are recorded below. Other keyboard scenarios need their named panel fixtures and a full feature rerun. |
| 1 | `features/application/ribbon-display/ribbon-display.feature` | In progress | The command-panel tab-switch scenario passes on the recorded source. Revalidate the remaining scenarios, including appearance and modeling-preview cases. |
| 1 | `features/application/workspace-layout/workspace-layout.feature` | Needs rerun | Source changed since the inventory checkpoint. Review the correction and rerun affected native scenarios. |
| 2 | `features/sketch/pattern-rectangular/scenario.feature` | In progress | Rectangular patterns can be created and edited in the native UI, but directions are fixed to the sketch axes. Implement line-reference selection and associativity, then run the affected native scenarios. |
| 2 | Other `features/sketch` | In progress | The native sketch gate runs only selected folders. Inventory and run the remaining folders before reporting total coverage. |
| 3 | `features/solid-model` | In progress | Implementations exist, but this area has not been assessed against the current complete source. Assess each feature after the application and sketch priorities; no total execution result is recorded. |

## Evidence records

### Quick Access Appearance Browser, 2026-11-13

- On source SHA-256
  `e7546a712fe3f42b2c6f9885e47292b69cb3ef0ebe3105a9ad9a26bbed40bbbd`,
  unchanged before and after, the native Quick Access run passed 11 of 12
  scenarios and 122 of 123 steps. The remaining scenario stopped at Refine
  Appearance before its Appearance Browser check. The benchmark returned
  failure. Application base `81e309810` plus the browser changes; Linux,
  Xvfb and Lavapipe; Martensite revision
  `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
- Ran `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`
  with `OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios`,
  `OBK_BDD_FEATURE=quick-access-toolbar` and
  `OBK_BDD_REPORTS=/tmp/obk-bdd-qat-20261113-appearance`. A focused native
  test opens both asset browsers, assigns an appearance, closes its own window,
  and undoes the assignment. A GUI test covers the browser's key tip.

### Quick Access Materials, 2026-11-13

- On source SHA-256
  `e7546a712fe3f42b2c6f9885e47292b69cb3ef0ebe3105a9ad9a26bbed40bbbd`,
  unchanged before and after, the native Quick Access run passed 11 of 12
  scenarios and 122 of 123 steps. The remaining scenario stopped at Refine
  Appearance before it reached Materials. The benchmark returned failure.
  Application base `bb5844e43` plus the Materials changes; Linux, Xvfb and
  Lavapipe; Martensite revision `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
- Ran `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`
  with `OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios`,
  `OBK_BDD_FEATURE=quick-access-toolbar` and
  `OBK_BDD_REPORTS=/tmp/obk-bdd-qat-20261113-materials`. A native pointer test
  assigns a catalog material, undoes it, and removes and restores the shortcut.
  A GUI test checks the key tip and its disabled state during a transaction.

### Quick Access Select, 2026-11-13

- On source SHA-256
  `e7546a712fe3f42b2c6f9885e47292b69cb3ef0ebe3105a9ad9a26bbed40bbbd`,
  the native Quick Access run passed 11 of 12 scenarios and 122 of 123 steps.
  The remaining scenario passed its Select check and stopped at the missing
  Refine Appearance choice. The benchmark returned failure. The source hash was
  checked during and after the run. Application base `6b46442e4` plus the
  Select changes; Linux, Xvfb and Lavapipe; Martensite revision
  `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
- Ran `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`
  with `OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios`,
  `OBK_BDD_FEATURE=quick-access-toolbar` and
  `OBK_BDD_REPORTS=/tmp/obk-bdd-qat-20261113-select`. The directory holds the
  reports and log. Native pointer tests cover the selection menu and an assembly
  component pick. An owner test covers part face picks and Measure priority.
  Edge and body picking remain outside this selection menu's current support.

### Quick Access Update, 2026-11-13

- On source SHA-256
  `e7546a712fe3f42b2c6f9885e47292b69cb3ef0ebe3105a9ad9a26bbed40bbbd`,
  unchanged before and after, the full native Quick Access run passed 11 of 12
  scenarios. The remaining scenario passed its Update check and stopped at
  missing Select. The benchmark returned failure. Application base
  `433c0bc85` plus the Update changes; Linux, Xvfb and Lavapipe; Martensite
  revision `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
- Command: `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`,
  with `OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios`,
  `OBK_BDD_FEATURE=quick-access-toolbar`, and
  `OBK_BDD_REPORTS=/tmp/obk-bdd-qat-20261113-update`. That directory holds JSON,
  JUnit and the log. A focused native pointer test checked Update without an
  undo step. Owner tests checked healthy replay, failure reporting and recovery.
  At that run, the CLI host did not yet supply this evaluation owner. It now
  uses the shared replay operation and retains a result per open document.
  Native CLI script tests cover update, rebuild, freshness, multiple documents,
  failed-feature reporting and recovery.

### Quick Access Toolbar current-source run, 2026-11-13

- Source: `features/application/quick-access-toolbar/quick-access-toolbar.feature`,
  SHA-256 `e7546a712fe3f42b2c6f9885e47292b69cb3ef0ebe3105a9ad9a26bbed40bbbd`
  before and after. Application commit `3ea391ea3` plus the Quick Access
  Parameters changes; unrelated local `README.md` and `AGENTS.md` changes.
  Linux, Xvfb and Lavapipe; Martensite revision
  `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
- With `OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios`,
  `OBK_BDD_FEATURE=quick-access-toolbar` and
  `OBK_BDD_REPORTS=/tmp/obk-bdd-qat-20261113-verified`, ran
  `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`.
  Eleven of 12 scenarios passed; the remaining scenario stopped at its first
  missing choice, Update. The benchmark returned failure. The report directory
  contains the JSON, JUnit and log. Later choice checks did not run.
- A native pointer test opens Parameters from Quick Access, adds
  `quick_size = 10 mm`, closes the dockable window and removes the shortcut.
  It also checks that the same part stays active. The editor now starts below
  the ribbon; a captured frame confirmed that its input fields are visible.

### Document creation current-source run, 2026-11-13

- Source: `features/application/document-creation/document-creation.feature`,
  SHA-256 `e9a93d4c096090667952b8080a0cad01c48f480022a4b8ac72ae210607ca138d`
  before and after the run. Application commit `9b1fe4978`; unrelated local
  `README.md` and `AGENTS.md` changes were present. Linux, Xvfb, Lavapipe;
  Martensite revision `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
- With `OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios`,
  `OBK_BDD_FEATURE=document-creation`, and `OBK_BDD_REPORTS` set to
  `/tmp/obk-bdd-create-20261113-current`, ran
  `xvfb-run -a -s '-screen 0 1600x1000x24' cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture`.
  All 39 scenarios ran: 14 passed; 25 stopped at undefined steps. The test
  returned failure. JSON, JUnit, and the log are in that report directory.
  A run after the binding updates is required for current results.
- A second full run with the first wording bindings passed 22 of 39 scenarios;
  17 stopped at undefined steps. Source hash was unchanged. Reports:
  `/tmp/obk-bdd-create-20261113-bindings`. More bindings were added while
  that run was in progress, so it does not establish the current count.
- The full run with all current wording bindings passed 26 of 39 scenarios
  (66.7%); 13 stopped at undefined steps, and the benchmark returned failure.
  The source hash was unchanged before and after the run. The application was
  `9b1fe4978` with the three document-creation and project-dialog binding
  files modified. The same native command and environment were used, with
  `OBK_BDD_REPORTS=/tmp/obk-bdd-create-20261113-final`; JSON, JUnit and log
  are there. Remaining work includes the named template fixtures, keyboard
  navigation setup, populated drawing layouts and their update behavior.
- A further full run on the same source hash passed 27 of 39 scenarios
  (69.2%); 12 stopped at undefined steps. The test returned failure. It used
  application commit `18f721807` with the gallery setup, inch-template action,
  and active-sketch setup bindings modified. The same native command and
  environment were used, with `OBK_BDD_REPORTS` set to
  `/tmp/obk-bdd-create-20261113-gallery`. The setup now reaches the named
  template checks. The active-sketch scenario asks for a `Create New File`
  title; the current chooser title is `New document`.

### Document creation focused scenarios, 2026-11-13

- On source SHA-256
  `e9a93d4c096090667952b8080a0cad01c48f480022a4b8ac72ae210607ca138d`,
  the native drawing-template creation scenario passes all 10 steps. The
  check verifies the sheet, title block, browser, tab and enabled Base View
  command. The previous binding incorrectly treated Base View as unavailable.
- On the same source, the template grouping and assembly list-view scenarios
  pass all 10 and 11 steps respectively with native pointer input. These are
  focused results; the complete creation feature needs a new run before its
  pass count can be updated. Logs: `/tmp/obk-create-drawing-current.log` and
  `/tmp/obk-create-gallery-0.log`, `/tmp/obk-create-gallery-1.log`.
- The presentation creation and template-group collapse scenarios each pass
  all 10 steps through the native UI on that source. The presentation check
  includes the Insert dialog, its cancellation and reopening, the scene browser,
  and both workspace panels. Logs: `/tmp/obk-create-presentation.log` and
  `/tmp/obk-create-collapse.log`.

### Drawing browser and saved views, 2026-11-13

- The full Open feature at source SHA-256
  `d7965f270ecc21d99f2ed9a0837f17a1996203fccf4d4d1966e7876ebb01464c`
  ran on application commit `36a505afd` under Linux, Xvfb and Lavapipe.
  It passed 26 of 27 scenarios and 261 of 262 executed steps. The express
  assembly scenario stopped at its missing component fixture; none of its later
  checks ran. The benchmark returned failure as designed. The run used an
  unchanged copy of this feature to isolate the unrelated Home parse error.
  Log: `/tmp/obk-open-full-current.log`.
- A native regression now checks that double-clicking a selected deferred
  drawing shows the deferred message and displays the saved view without its
  source model. The normal double-click and deferred-open scenarios also pass
  after this fix: 9 and 15 steps respectively.

- Historical deferred-open assessment on the same feature source: the focused native
  scenario reaches the information message and passes ten steps, then fails
  because acknowledging it does not open the drawing. The dialog sends
  `deferContent`, whose API contract requests a reference stub rather than
  deferred drawing updates; the host rejects that option. A separate drawing
  defer contract and persisted view geometry are needed. The current test
  fixture contains a saved view definition and model, not a saved projection;
  extend it when saved projections exist. Local report:
  `/tmp/obk-deferred-baseline2`.
- The package layer now has a versioned `presentation/drawing-views-v1` stream
  for saved sheet-space view geometry. It checks the complete drawing recipe, exact
  sheet/view coverage, duplicate names and finite line coordinates. A changed
  drawing recipe removes the old stream; ordinary cache compaction keeps it.
  The Go YAML codec retains named streams. The
  full persistence suite passed 382 tests after correcting a stale sketch
  projection version assertion that also failed at the prior commit.
- The session save target can stage a saved-view snapshot in the same atomic
  candidate as the drawing recipe. Tests cover Save As, Save Copy As,
  save-on-close, and refusal of a stale snapshot without changing the file.
  The desktop's shared host now captures the projected lines for Save, Save As,
  Save Copy As, save-on-close and batch saves. Capture failure refuses that
  save and a batch continues with the other items. The desktop and CLI now use
  the same Martensite projection service and document host. CLI Save As and
  script saves capture referenced views, and native file paths retain their OS
  bytes. A missing source prevents a drawing save. A host without the injected
  projection service still refuses to save a drawing with views.
  The file workspace can now retrieve the validated last-saved view geometry
  after save or reopen, even when the live drawing has since changed. A session
  can freeze a matching saved presentation and the desktop projection selects its
  active-sheet lines without evaluating the model. The open dialog now uses a
  distinct deferred drawing action and disables view creation. The focused
  native BDD scenario passes all 15 steps with a Martensite-generated saved
  projection. A drawing without a saved presentation is refused before its
  document is registered. Save, Save As and Save Copy preserve frozen views
  without the model; a changed drawing recipe is refused instead of saving a
  stale presentation. Changing defer status is not yet implemented. The full current
  application benchmark also reports a parse error in the separately updated
  home-recent-documents feature; the focused run used an unchanged copy of the
  document-opening feature. Report: `/tmp/obk-deferred-isolated.log`.
  Seven native drawing tests passed, including real projected-line storage and
  a mixed-success batch; the 416 session lifecycle and 726 API query tests passed.
- The full Open feature, SHA-256
  `d7965f270ecc21d99f2ed9a0837f17a1996203fccf4d4d1966e7876ebb01464c`,
  passed 23 of 27 scenarios and 227 of 231 executed steps on Linux with Xvfb
  and the configured Vulkan runtime. Four scenarios stopped at undefined
  steps, so their later checks did not run. Local reports are in
  `/tmp/obk-open-drawing-suite`.
- The focused “Open two selected drawings together” scenario now has native
  fixtures with two closed drawings, real saved projections and an available
  shared source. It checks the thumbnail selection, both drawing tabs and
  rendered sheets, and preservation of the original tab. On the same feature
  SHA, all 11 steps pass under Xvfb; report: `/tmp/obk-two-drawings.log`.
- The focused “Open a different drawing by double-clicking its row” scenario
  passes all nine steps with an active assembly and two closed drawings. It
  checks the second drawing's saved view, its rendered sheet and tab, and the
  retained assembly tab. Report: `/tmp/obk-double-drawing.log`.
- Express assembly loading remains unimplemented: the Rust host has no
  assembly occurrence loading or component rendering path. The Open dialog
  now reports that limit and refuses Open, Enter and double-click while Express
  is selected, instead of silently opening in full mode. Native input tests
  cover those routes and the drawing-option reset scenario still passes all
  11 steps. Go's `model/occurrence` is the reference for component instances.
- Until occurrence reconstruction is implemented, typed loading and template
  creation refuse Go assembly recipes with occurrences or occurrence patterns.
  This prevents a populated assembly from being presented or saved as an empty
  model. Envelope reading still retains the original data, and empty assemblies
  remain supported. This is a named decline, not assembly implementation.
- The package layer can now inspect Go-shaped occurrence recipes without
  opening component files. It preserves instance flags, component references,
  virtual component fields and child transforms; it validates finite 16-cell
  matrices and resolves an omitted placement to identity. Tests cover shared
  references, virtual data, YAML round-trip and malformed transforms. This
  reader does not yet attach occurrences to document history, resolve their
  references or render them, so the populated-assembly open guard remains.
- A fixture generated by the Go host at commit `aa8297d1b` now verifies two
  placements of one shared component, distinct transforms and instance flags
  through the Rust package reader and re-encoding. The generator is retained
  with the fixture. Typed open still refuses this populated package until the
  reference and rendering paths exist.
- The package layer can prepare validated occurrence changes while preserving
  the other Go package fields. It refuses changes to component references or
  edits with unported dependent patterns or features; a document owner must
  eventually perform those operations with the reference graph. This does not
  enable populated assembly opening or claim an atomic save by itself.
- Occurrence recipes are now owned by document history, included in snapshot
  change comparison, and captured by the existing atomic save target. Tests
  cover undo/redo, a saved placement edit, empty assembly creation and the
  populated-template decline. The workspace all-target compile check, 419
  session lifecycle tests, 39 selected persistence tests, and the native
  parameter-driven cube journey passed. Populated assembly opening stays
  disabled until component resolution and rendering are in place.
- Assembly recipes now provide distinct external component references in
  first-placement order, including suppressed placements and excluding virtual
  components. Relative reference paths use the owning document's folder through
  a shared document-owner operation; the drawing loader uses the same operation.
  Tests cover duplicate references, path errors and the native lifecycle of two
  drawings sharing one relative model. Assembly reference loading is still open.
- The file owner now tracks multiple authored references per document in one
  cycle-checked graph shared by drawings and future assemblies. Shared models
  stay open until their last owner closes, and Close All uses an owner-first
  order for nested references. Tests cover cycles, nested order, invalid target
  types, shared ownership and the existing native drawing lifecycle. Assembly
  files are still guarded until their reference loader and renderer are ready.
- Reference failures are now stored per authored component rather than per
  drawing. A failed component leaves other bindings intact, and a successful
  rebind clears only its own error. The drawing path uses the same error owner.
  Session tests cover independent failures, recovery and cleanup; the native
  missing-model drawing regression also passes.
- The host can resolve distinct assembly component references in the background,
  report failures separately, retry missing files, and release unused background
  documents after an edit. Drawings share the same loading path. Host tests
  cover shared sources, recovery and invalid target cleanup; native drawing
  tests cover shared and missing sources. This resolver is not yet connected to
  populated assembly opening, which remains guarded pending rendering support.
- Viewport scenes can now be copied through a checked row-major occurrence
  placement. Tests cover a saved occurrence's rotation and translation, source
  preservation, and invalid or overflowing placements. Live opening remains
  guarded.
- The desktop scene owner can now compose resolved component scenes, including
  nested placements and suppression, and refresh after a component edit. Tests
  create an in-memory assembly owner and check placement, refresh and missing
  references. Populated assembly files still cannot open: reference loading,
  scene composition and the user-facing failure path need integration and a
  native end-to-end test.
- A scene that cannot resolve or place a component now reports the occurrence
  and cause in the desktop scene status while showing the other components,
  instead of terminating the frame loop.
  The same failure is not published every frame, and a resolved reference
  triggers a new scene. Tests cover the failure and recovery path; populated
  assembly file opening remains guarded.
- Subsequent integration enables opening saved assemblies with supported rigid
  occurrences. It resolves nested component files in the background and shows
  missing references in scene status. A native mouse-driven open displayed a
  referenced part, and a separate test checked a missing part. The captured
  frame shows that the model browser does not yet list occurrences. Assembly
  features, patterns, adaptive, flexible and substitute occurrences remain
  refused on open. The earlier open-guard notes describe the prior state.
- The model browser now lists assembly occurrences in their authored order,
  including nested components, suppressed rows and missing-reference markers.
  Browser search keeps the parent of a matching nested occurrence. Native tests
  cover opening and searching the tree; selection and component commands are
  not yet implemented for those rows.
- Occurrence rows can now be selected with the mouse. The owner checks a path
  scoped to each assembly generation, so a stale row cannot select a different
  occurrence after an edit. Tests cover duplicate names, clearing selection,
  stale paths and mouse selection of nested rows. Component commands and
  selection highlighting in the viewport remain open.
- A selected occurrence now tints only its placement in the viewport; selecting
  a subassembly tints its visible children. Scene tests check independent
  placements and clearing, and a native mouse test checks the pixel colors of
  two separated placements. Component commands and viewport picking remain open.
- Assembly viewport clicks now query the referenced component B-reps through
  Martensite after transforming the sight line into each placement. The owner
  compares hit distances in one coordinate system, rejects exact placement
  ties and reports invalid transforms. Tests cover translated and scaled
  placements, a nested mouse pick and a miss. Face-level assembly references
  and occurrence editing commands remain open.
- Top-level occurrence rows now offer Suppress and Unsuppress through one
  generation-checked host operation. A native mouse and keyboard test covers
  suppression, undo, redo, restoration, save and reopen; framebuffer checks
  confirm that the suppressed placement disappears. Host tests cover stale
  paths, vetoes and persistence. The public `assembly.suppress` wire method
  and stable occurrence session IDs are not yet implemented; nested rows must
  be edited in their owning assembly.
- The canonical `assembly.occurrences` and `assembly.suppress` methods now
  share the host's suppression operation with the browser. Opened occurrences
  receive session IDs that stay with them through edits and undo/redo, including
  reordering; the IDs are not written into Go packages. Querying an unresolved
  component reports an error. Placement creation and an allocator for new
  occurrence IDs are still needed, and nested IDs belong to their own assembly.
- Document history now reserves occurrence IDs outside undo snapshots. It
  refuses zero, duplicate and unreserved IDs, and keeps reservations after
  abort and undo. Preview uses a private copy of the reservation mark. This
  supplies the allocator for placement; the placement command and its
  reference-graph and save integration remain open.
- The drawing browser now lists the authored sheets, borders, title blocks and
  views, and browser search uses those same rows. Unit tests cover multiple
  sheets, a missing border and searching for a view.
- The focused Open scenario creates a saved part and a drawing that references
  it, opens the drawing with keyboard and mouse, checks the rendered projection
  and browser row, and retains the original part tab. The referenced part is
  opened in the background for this fixture. The native screenshot was inspected
  at `/tmp/obk-open-drawing-green/Open-a-saved-drawing-alongside-an-existing-part.png`.
- Opening a drawing now loads a closed referenced part in the background. Native
  tests cover reopening, relative paths, two drawings sharing a source, Close
  All, reuse of an open part, and an unavailable source. The focused BDD case
  passes without preopening the referenced part. The document API query suite
  passed 726 tests. Reference changes now replace the background owner; a native
  test checks keyboard undo and redo, a missing replacement, recovery with
  API undo, and transaction abort. It checks the projection and retained paths.
  The closed-source screenshot was inspected at
  `/tmp/obk-drawing-closed-bdd/Open-a-saved-drawing-alongside-an-existing-part.png`.

### Open dialog selection, 2026-11-13

- Source: `features/application/document-opening/document-opening.feature`,
  SHA-256 `d7965f270ecc21d99f2ed9a0837f17a1996203fccf4d4d1966e7876ebb01464c`.
  On Linux with Xvfb and the configured Vulkan runtime, the full native run
  passed 22 of 27 scenarios and 219 of 224 executed steps. Five scenarios
  stopped at undefined steps; their later checks did not run. Reports are in
  `/tmp/obk-open-final2-20261113`; they are local and not durable.
- The drawing-option reset and file context-menu scenarios now pass. A click
  in empty list space dismisses the context menu and clears the selection,
  while retaining the file name and preview. A native test also checks the
  menu's Properties command, and the Save As selection scenario passes.
  The 48 GUI save tests, formatting and Clippy passed. The feature run used
  equivalent code before a later Clippy-only `?` rewrite.

### First-save isometric preview, 2026-11-13

- Source: `features/application/document-saving/document-saving.feature`,
  SHA-256 `ee58ca1d01af2e1db700be89fa237cee19f7d42c477fbe6c8f5f62d07683cbd2`.
  The full native feature run passed 23 of 23 scenarios and 218 of 218 steps
  on Linux with Xvfb and the configured Vulkan runtime. Reports are in
  `/tmp/obk-save-current-20261113`; they are local and not durable.
- A separate native test creates a constrained rectangle and extrudes it,
  checks the shaded preview and unchanged active camera, saves the part,
  checks its stored picture, and reopens the solid. The preview is rendered
  offscreen and an error is shown if capture fails. GUI save tests passed;
  the preview lighting tests cover face shading and degenerate geometry.
- The owner scene is refreshed before an isometric capture. A focused test
  checks that scene publication comes before preview capture; the feature run
  above includes that change.

### Save As selection and save scenarios, 2026-11-13

- Source: `features/application/document-saving/document-saving.feature`,
  SHA-256 `ee58ca1d01af2e1db700be89fa237cee19f7d42c477fbe6c8f5f62d07683cbd2`.
  Application: `5e3f27e4f` plus the Save As selection, redraw, and binding
  changes in this work; Martensite uses the workspace lockfile.
- Linux, Xvfb at 1600 × 1000, configured Vulkan runtime, `XMODIFIERS` unset.
  The full native feature run passed 22 of 23 scenarios and 211 of 212
  executed steps. The remaining scenario stopped at an undefined unsaved-solid
  fixture, so its preview checks did not run. Reports are in
  `/tmp/obk-save-final-20261113`; they are local and not durable.
- Selecting a Save As folder now highlights it without changing the file
  name; double-click opens it, and clicking empty list space clears the
  highlight. Details-view double-clicks use the row's actual hit bounds, and
  a selection change requests another frame to show its highlight. The
  selection and folder-navigation native scenarios passed. The image chooser
  cancellation scenario passed after a follow-up guard for its redraw state.
- The GUI Save tests passed, as did a focused native saved-sketch reopen and
  an Open-dialog saved-part inspection. The full Open feature, source SHA-256
  `d7965f270ecc21d99f2ed9a0837f17a1996203fccf4d4d1966e7876ebb01464c`,
  passed 20 of 27 scenarios and 209 of 216 executed steps; seven cases stopped
  at undefined steps. Its local reports are in `/tmp/obk-open-final-20261113`.
  The full runs preceded the image-chooser redraw guard, which was checked
  separately through the image-chooser cancellation scenario.
