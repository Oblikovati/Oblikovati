# Go center point baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

On Linux, `go test ./app -run '^(TestCenterPointModeMarksNewPoints|TestPointToolPlacesPoint)$' -count=1` passed. The first session test commits a point with Center Point enabled and checks its role. The second checks point creation and expects the Go tool to deactivate after placement. Neither test checks native marker rendering or preservation of an existing regular point. The selected Rust scenario checks those separately; it does not establish visual parity with Go.
