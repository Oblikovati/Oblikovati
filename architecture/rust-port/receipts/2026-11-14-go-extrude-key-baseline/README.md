# Go extrusion key reference

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

On Linux, `go test ./app -run '^(TestExtrudeViaCommand|TestCommandInputCommitsCustomAlias|TestSingleLetterAliasHasNoDefaultChord)$' -count=1` passed. The Go command registers `E` as an alias and `Shift+E` as a chord. Its keymap test deliberately does not treat a bare single-letter alias as a default chord. The pinned scenario specifies a single `E` press; its Rust acceptance is a benchmark behavior and does not prove keyboard parity with Go.
