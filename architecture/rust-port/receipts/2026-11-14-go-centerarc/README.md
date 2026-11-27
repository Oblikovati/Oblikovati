# Go center-point arc reference

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/sketch_variant_tools.go` creates an arc from a center, start, and end.

On Linux, `go test ./app -run '^TestCenterPointArcTool$' -count=1` passed. It checks that three clicks produce one arc. It does not inspect the selected arc orientation, the native Rust ribbon, or the rendered result.
