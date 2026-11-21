# Clear multiple selected bodies

On the task branch based at `687d7dea0`, browser Ctrl-click selected two exact
body keys. Clear and Cancel left the three body overrides intact. Clear and OK
removed the selected two, left the third intact, and one Undo and one Redo
restored and cleared both. The test saved and reopened the document, retaining
the third override and the part material and appearance. The capture shows the
two selected browser rows and the confirmation. The test bodies overlap, so
the capture does not establish a distinct 3D highlight for each body.

A separate owner test checks one history entry, an unrelated named face
override, changed-selection refusal, and forged-body refusal. Session tests
check replace, Ctrl/Shift toggle, key identity and retiring body picks when the
root or a sketch is selected. The existing GUI library suite passed 498 tests;
the native Clear group passed three tests. Strict session, GUI and head Clippy,
formatting and the source-size guard passed. Logs retain the test output.

Go reference: `app/session_input.go` applies replace or Ctrl/Shift toggle to an
ordered selection; `app/style_assign.go` records body style changes as edits.
The pinned Quick Access feature checks the Clear menu item, but does not specify
a multi-target Clear scenario. This native regression uses frame-loop input,
not operating-system input. Viewport body picks still replace selection;
viewport multi-select, face selection, assembly occurrences, and the full gate
remain open under #3962.
