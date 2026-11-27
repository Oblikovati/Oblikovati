# Go arc cancellation reference

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/sketch_geometry_tools.go` resets the arc tool when cancelled.

On Linux, `go test ./app -run '^(TestEscapeCancelsToolMidOperation|TestArcToolDrawsArc)$' -count=1` passed. The cancellation test uses a line, while the arc test covers creation. These tests do not directly exercise Escape after two arc points or the native Rust ribbon.
