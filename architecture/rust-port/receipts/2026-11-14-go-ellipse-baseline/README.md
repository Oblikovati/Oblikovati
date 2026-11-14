# Go ellipse baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

On Linux, `go test ./app -run '^TestEllipseToolDrawsEllipse$' -count=1` passed. The session test creates an ellipse with three clicks and checks that both radii are positive. It does not check cancellation, undo, redo, rotated axes, or native rendering. The Rust native receipts cover the selected pinned steps and inspected final captures; they do not establish visual parity with Go.
