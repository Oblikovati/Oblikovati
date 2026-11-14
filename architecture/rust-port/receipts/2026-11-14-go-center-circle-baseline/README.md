# Go center-point circle baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/sketch_geometry_tools.go` owns the Go center-point circle tool.

On Linux, `go test ./app -run '^(TestCircleToolDrawsCircle|TestRectangleAndCirclePreview)$' -count=1 -v` passed. These tests check a synthetic two-click circle with positive radius and a closed preview. They do not verify native dropdown input, same-position rejection, Undo/Redo, or rendered pixels. The Rust native receipts cover the selected pinned scenario steps; they do not establish Go visual parity or all circle scenarios.
