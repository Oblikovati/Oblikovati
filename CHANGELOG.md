# Changelog

## [Unreleased]

### Fixed

- Made the default ribbon display control toggle between full and tabs-only modes.

- Made Precise Input show coordinates and the triad for the selected reference mode.

- Resolved API dimension Drive expressions once, as in the Go host, while keeping desktop formulas live.

- Kept selected sketch geometry at the requested transform, and refused moves blocked by constraints.

- Kept geometric constraint tools active for repeated selections and restored Coincident prompts.

- Preserved sketch selection when opening a point context menu. (#3856)

- Disabled sketch marking-menu commands while a transaction is open. (#3856)

- Made Escape from the ribbon target selector discard the pending customization. (#3856)

- Kept origin-plane labels inside the clickable workspace when starting a sketch. (#3857)

### Added

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

### Fixed

- Included User Commands in the ribbon panel visibility controls. (#3856)

- Added a missing-file warning and filename recovery to ribbon settings import. (#3856)

- Fixed ribbon customization to keep display options per command and restore them after a staged removal. (#3856)

- Kept reversed ribbon customization edits pending until Apply or Cancel. (#3856)

- Fixed Escape in ribbon customization to discard pending edits without cancelling the active sketch. (#3856)

- Show the model's reason when a sketch dimension value is refused. (#3855)

- Reject driving dimension values that fail the sketch constraint's target
  checks, including a zero radius, before saving the edit. (#3855)
