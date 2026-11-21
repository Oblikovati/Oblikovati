# Go validation reference

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

On Linux, `go test ./app -run '^(TestExtrudeToolNeedsProfileAndDistance|TestPointToolPlacesPoint|TestSketchCreationIsUndoable)$' -count=1` passed. The extrusion test checks an absent profile and a positive distance; it does not isolate zero distance with a selected profile. `app/extrude_tool.go` includes a zero-distance admission check. The other two tests cover generic point creation and sketch-creation history; they do not verify invalid Precise Input text or point undo/redo through the native window. Go has a different Point-tool lifecycle. The Rust native scenarios and captures are separate evidence, not a claim of visual parity.
