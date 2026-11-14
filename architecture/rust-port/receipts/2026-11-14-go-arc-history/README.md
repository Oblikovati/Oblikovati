# Go arc history reference

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/sketch_geometry_tools.go` creates three-point arcs and
`app/sketch_undo_test.go` tests sketch creation history.

On Linux, `go test ./app -run '^(TestSketchCreationIsUndoable|TestArcToolDrawsArc)$' -count=1` passed. These tests check sketch creation undo and arc creation separately; they do not test this arc's undo/redo with surrounding geometry or the native Rust window.
