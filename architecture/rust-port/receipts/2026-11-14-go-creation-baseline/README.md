# Go document creation baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`head/ui/chrome_menubar.go` invokes `Session.NewPart` and `Session.NewAssembly`
directly; this checkout has no matching template chooser in `head/ui`.
`app/session.go` creates and activates each document type.

On Linux, `go test ./app -run '^(TestNewAssemblyCreatesActiveAssemblyDocument|TestNewPartCommandSwitchesRibbon)$' -count=1` passed. The first test checks the active assembly, its realized content, and its name. The second checks part activation and the part ribbon. Neither test checks the scenario's template chooser, pointer and keyboard interactions, or browser presentation. The native Rust scenario receipts provide evidence for those scenario-specific checks; they do not establish visual parity with Go.
