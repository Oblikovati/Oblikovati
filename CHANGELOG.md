# Changelog

## [Unreleased]

### Added

- Added immediate placement and value editing for sketch dimensions. Cancel keeps
  the measured dimension, and Undo treats placement and value edits as separate
  steps. (#3855)

- Added Left and Right navigation for feature branches in the model browser. (#3856)

- Added Up and Down navigation for visible rows in the model browser’s Features branch. (#3856)

- Added a Zoom Selected toolbar for selected model faces. (#3856)

- Added arrow-key view rotation while ribbon key tips are visible. (#3856)

- Added keyboard access to compact and side-docked ribbon groups for part documents. (#3856)

### Fixed

- Reject driving dimension values that fail the sketch constraint's target
  checks, including a zero radius, before saving the edit. (#3855)
