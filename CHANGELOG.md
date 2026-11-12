# Changelog

## [Unreleased]

### Added

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

- Fixed ribbon customization to keep display options per command and restore them after a staged removal. (#3856)

- Kept reversed ribbon customization edits pending until Apply or Cancel. (#3856)

- Fixed Escape in ribbon customization to discard pending edits without cancelling the active sketch. (#3856)

- Show the model's reason when a sketch dimension value is refused. (#3855)

- Reject driving dimension values that fail the sketch constraint's target
  checks, including a zero radius, before saving the edit. (#3855)
