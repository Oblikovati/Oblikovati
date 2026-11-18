# Changelog

## [Unreleased]

### Added

- Enabled the Rust desktop application on Windows through a native Win32 window and Martensite Vulkan rendering.
- Enabled isolated Lua scripting on Windows in the desktop and CLI.

- Added analytic face properties from the selected-face context menu. (#4053)

- Show the type and highlight of a sketch entity under the pointer in Ready. (#4043)

- Show the picked sketch entity type and sketch name near the pointer. (#4042)

- Added Move and Measure pointer feedback for sketch curves, points, and midpoints. (#4034)

- Added transient candidate feedback for sketch constraint and dimension tools. (#4034)

- Added an experimental offline sandbox build probe for native BDD investigations. (#3978)

- Added an experimental committed-source exporter for native BDD build investigations. (#3978)

- Added a native scenario receipt command that checks its artifacts before publication. (#3865)

- Added a diagnostic mode to report native BDD step gaps after failures. (#3863)

- Added browser rows and an editor for saved rectangular sketch patterns.

- Added a rectangular sketch-pattern dialog with live preview and cancel.

- Added rectangular sketch-pattern creation through the typed API.

- Added previewable and undoable rectangular pattern edits that retain existing member identities.

- Added saved rectangular sketch-pattern records with preview and undoable creation.

- Added parameter-driven spacing for linked sketch copies and recompute.

- Linked translated sketch copies to their seed points and circle radii through solve and history.

- Added saved structural point links for associative sketch-pattern solving.

- Added rectangular sketch-pattern layouts with fitted spans and suppressed instances.

- Added atomic sketch-copy batches and previews as a foundation for patterns.

- Added rotated sketch-copy previews and undoable copy operations.

- Kept Finish Sketch available on document ribbon tabs while a sketch is being edited.

- Added context menus for unavailable commands in expanded ribbon panels.

- Added the optional Parameters panel to the Sketch ribbon.

- Added document unit labels to Precise Input coordinates.

- Added Copy mode to sketch Move, with preview and undo/redo.

- Added precise coordinates and triad controls to sketch Move.

- Added multi-entity selection and single-selection optimization to sketch Move.

- Added single-entity sketch Move with a solved preview, canvas points, and undo/redo.

- Added undoable geometric relation removal from sketch curve context menus.

- Added repeat of sketch constraints to the sketch context menu. (#3856)

- Added selection-aware Delete to the sketch context menu. (#3856)

- Added sketch marking-menu overflow command editing. (#3856)

- Added saved sketch marking-menu layouts and slot customization. (#3856)

- Added sketch-ready aliases for Undo, Redo, Save, Home, and Previous View. (#3856)

- Added saved keyboard shortcuts and sketch command aliases to the Customize dialog. (#3856)

- Added ribbon location information for the sketch visibility command. (#3856)

- Added the View Tab command category to ribbon customization. (#3856)

- Added keyboard switching between ribbon customization targets. (#3856)

- Added keyboard navigation and Escape handling to the ribbon command-category list. (#3856)

- Added the Macros category to ribbon customization for sessions with no macros. (#3856)

- Added keyboard navigation and scrolling for available ribbon commands. (#3856)

- Added name sorting to the available ribbon commands while preserving selection. (#3856)

- Added expanded-panel placement for custom ribbon commands. (#3856)

- Saved applied part ribbon customization in user preferences across restarts. (#3856)

- Added XML import and export for staged part ribbon customization. (#3856)

- Added Ctrl+A selection and a title-bar close control to ribbon customization. (#3856)

- Added keyboard selection and transfer of multiple ribbon commands. (#3856)

- Added sketch tools and constraints to the ribbon customization command list. (#3856)

- Added double-click transfer of ribbon user commands. (#3856)

- Added session-local customization of part ribbon user commands. (#3856)

- Added typed API placement of existing sketch block definitions with checked
  position, rotation, scale, and undo/redo. (#3856)

- Added viewport display for placed sketch block definitions, including their
  rotation, scale, and instance selection color. (#3856)

- Added saved sketch block instance identities and placements, with typed
  instance listing and protection against deleting a definition in use. (#3856)

- Added saved local geometry for sketch block definitions, including captured
  shared points and curve roles. (#3856)

- Added typed API operations to create, list, and delete empty sketch block
  definitions, with undo and save/reopen support. (#3856)

- Added the Sketch Create expansion with access to Precise Input while a sketch
  command accepts points. (#3856)

- Added the Sketch tab for part documents and its Constrain expansion. (#3856)

- Added immediate placement and value editing for sketch dimensions. Cancel keeps
  the measured dimension, and Undo treats placement and value edits as separate
  steps. (#3855)

- Added Left and Right navigation for feature branches in the model browser. (#3856)

- Added Up and Down navigation for visible rows in the model browser’s Features branch. (#3856)

- Added a Zoom Selected toolbar for selected model faces. (#3856)

- Added arrow-key view rotation while ribbon key tips are visible. (#3856)

- Added keyboard access to compact and side-docked ribbon groups for part documents. (#3856)

### Changed

- Reused the current source part result when creating a drawing base view. (#4079)

- Reused current part results when typed Fillet and Chamfer commands resolve selected edges. (#4079)

- Retained current part replay results per open document in the desktop. (#4079)

- Reused unchanged sketch replay results during part updates to avoid a second full replay. (#4079)

- Separated selected-face context commands from the radial marking menu. (#4055)

- Enabled analytic area measurement from a selected face context menu. (#4052)

- Enabled Chamfer from the default Part marking menu. (#4046)

- Kept selected input edges visible during a Fillet preview. (#4045)

- Allowed the Fillet tool to start before an edge is selected. (#4045)

- Enabled Fillet radius editing from the model browser with preview and cancel. (#4045)

- Enabled certified Fillet radius edits through the feature API, including named parameters and undo/redo. (#4045)

- Reduced repeated projection of constraint glyphs in an active sketch. (#4035)

- Enabled named-edge Fillet preview and creation in the Part viewport, with undo, redo, and save/reopen support. (#4045)

- Refresh camera depth when an edited model would be clipped. (#4044)

- Preserve body-relative face orientation in prism presentation. (#4044)

- Reduced repeated sketch point projection work during pointer interaction and context picking. (#4035)

- Discarded closed document-window preview images and stale pending updates. (#3987)

- Added type-preserving integer, number, boolean, and byte property editing to iProperties. (#3958)

- Added an optional locked-registry comparison to the offline BDD build probe. (#3978)

- Added an optional committed-source comparison to the offline BDD build probe. (#3978)

- Enabled viewport selection of visible work planes for sketch and offset commands. (#3984)

- Displayed visible authored work planes with a translucent fill and themed border in part views. (#3981)

- Enabled creation of offset work planes from the 3D Model ribbon, with document-unit validation and undo/redo. (#3977)

- Enabled analytic length measurements for named solid edges through the Analysis API. (#3967)

- Saved assemblies can include a qualified display cache; save warnings explain when it cannot be created. Express loading remains unavailable. (#3949)

- Exposed a shared assembly presentation traversal with explicit selection styling for the live scene; Express loading remains unavailable. (#3949)

- Extended staged legacy part replay to face-offset planes with independently witnessed side lineage; production precision qualification remains required. (#3882, #3928)

- Added a shared resolver that matches witnessed Go profile boundaries to current certified regions, with precision qualification required. Automatic Go profile witnessing remains open. (#3882, #3928)

- Added conservative physical-unit labels to raw sketch residual rows, with unknown units retained where the equation is ambiguous. These labels do not certify spatial or angular bounds. (#3928)

- Exposed source-attributed raw sketch residual rows and their convergence domains in solver diagnostics; these values are not certified spatial bounds. (#3928)

- Split sketch frame operations into `HeadSketchWork`; custom `HeadWork` owners must implement that interface. (#3860)

### Fixed

- Avoided computing the assembly producer identity for ordinary document commands. (#4081)

- Allow Fillet and Chamfer creation after model-state changes and reopening a part. (#4080)

- Preserved dock tabs when tool windows close, suspend, or reopen. (#4073)

- Replayed fillets and chamfers consistently in drawing, feature health, and extrusion preview. (#4074)

- Enabled Link Parameters cancellation without a source and restored the next dock tab on close. (#4073)

- Kept the remaining dock tab active when Precise Input closes from its menu. (#4073)

- Kept the remaining dock tab active when Create Block closes or completes. (#4073)

- Kept the remaining dock tab active when Navigation Wheel Options closes. (#4073)

- Rejected a selected-face Measure action when its face or document changed. (#4052)

- Started an associative sketch on the selected face from the Part marking menu. (#4050)

- Keep hover feedback in sync with a sketch changed before the next frame. (#4051)

- Restored Extrude from the Part marking menu for empty parts and finished sketches. (#4049)

- Listed and selected overlapping sketch point and curve candidates in Select Other. (#4041)

- Refused preference saves when YAML keys have ambiguous merge meaning, preserving the existing file. (#4007)

- Preserved document and Home tab order when scrolling and closing windows. (#3993)

- Kept off-screen document selection from highlighting a different visible tab. (#3992)

- Kept document tab previews clear of the tab at large UI scales and near window edges. (#3988)

- Cleared selected body and face appearance overrides together in one undoable edit. (#3962)

- Cleared multiple selected named face appearances in one undoable edit. (#3962)

- Cleared a selected named face appearance with Cancel and undo support, while keeping other overrides. (#3962)

- Kept Ctrl and Shift viewport body selections for Clear Appearance. (#3962)

- Kept the rectangle placement prompt while a zero-size preview is incomplete. (#3975)

- Excluded isolated task worktrees from source and documentation checks while retaining active nested modules. (#3971)

- Reopened existing extrusions for distance editing from the browser, with a cancellable candidate preview. (#3968)

- Edited extrusion distances without changing shared sketch parameters; literal distances can also be edited. (#3968)

- Kept sketch ribbon controls and the previous part tab available across sketch editing. (#3956)

- Refused legacy cap-program staging when qualified sketch precision evidence is unavailable. (#3928)

- Rejected ambiguous or forward legacy sketch and extrusion creation order before preparing a part candidate. (#3936)

- Decoded legacy work-plane face references with Go's unpadded URL-safe base64 format. (#3935)

- Showed the actual Part template material state, including unassigned and unresolved identities, in the chooser. (#3932)

- Reopened the Extrude tool with its configurable shortcut after cancelling a draft. (#3931)

- Refreshed extrusion previews after valid draft or parameter changes without creating features. (#3930)

- Validated extrusion distance drafts with the document owner, and retained rejected drafts for correction. (#3929)

- Resumed Extrude after its requested sketch finishes, with a 10 mm preview for a converged closed sketch, including a sketch with free variables. Sketch solve precision qualification remains open. (#3927, #3928)

- Cleared the save-path request after opening Save As, so cancellation leaves no stale error message. (#3888)

- Preserved the conversion guard of built-in model parameters when saving and reopening. (#3883)

- Made sketch creation and Finish Sketch separate undo and redo steps. (#3880)

- Started a sketch directly on a preselected origin plane. (#3879)

- Kept the ViewCube visible and interactive while editing a 2D sketch. (#3877)

- Enabled analytic area measurements for named solid faces through the Analysis API. (#3876)

- Kept toolbar arrow-key navigation from rotating the model on its first key press. (#3874)

- Lit model faces in the viewport and isometric preview so their orientations are visible. (#3873)

- Kept the model inside the document area when using Home or Fit with docked panels. (#3872)

- Selected a ribbon tab on the click that dismisses its context menu.

- Closed unpinned ribbon expansions when Escape dismisses their title menu.

- Kept expanded ribbon commands open while their docking menu is in use.

- Kept the ribbon docking submenu closed after Escape until it is opened again.

- Made the default ribbon display control toggle between full and tabs-only modes.

- Made Precise Input show coordinates and the triad for the selected reference mode.

- Resolved API dimension Drive expressions once, as in the Go host, while keeping desktop formulas live.

- Kept selected sketch geometry at the requested transform, and refused moves blocked by constraints.

- Kept geometric constraint tools active for repeated selections and restored Coincident prompts.

- Preserved sketch selection when opening a point context menu. (#3856)

- Disabled sketch marking-menu commands while a transaction is open. (#3856)

- Made Escape from the ribbon target selector discard the pending customization. (#3856)

- Kept origin-plane labels inside the clickable workspace when starting a sketch. (#3857)

- Included User Commands in the ribbon panel visibility controls. (#3856)

- Added a missing-file warning and filename recovery to ribbon settings import. (#3856)

- Fixed ribbon customization to keep display options per command and restore them after a staged removal. (#3856)

- Kept reversed ribbon customization edits pending until Apply or Cancel. (#3856)

- Fixed Escape in ribbon customization to discard pending edits without cancelling the active sketch. (#3856)

- Show the model's reason when a sketch dimension value is refused. (#3855)

- Reject driving dimension values that fail the sketch constraint's target
  checks, including a zero radius, before saving the edit. (#3855)

- Resolved viewport body colors from document and catalog appearances. (#3859)
