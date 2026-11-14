# Go three-point arc baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/sketch_geometry_tools.go` fits an arc through start, end, and a point on
the curve. `app/commands_sketch.go` registers the tool.

On Linux, `go test ./app -run '^(TestThreePointArcInterpretsItsClicksAsStartEndThrough|TestBothThreePointArcToolsAgreeOnPickOrder)$' -count=1` passed. The tests check the point order and agreement with the arc-slot tool. They do not exercise the Rust ribbon flyout or the native window.
