# Go document creation baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`head/ui/chrome_menubar.go` invokes `Session.NewPart` and `Session.NewAssembly`
directly; this checkout has no matching template chooser in `head/ui`.
`app/session.go` creates and activates each document type.

On Linux, `go test ./app -run '^(TestNewAssemblyCreatesActiveAssemblyDocument|TestNewPartCommandSwitchesRibbon)$' -count=1` passed. The first test checks the active assembly, its realized content, and its name. The second checks part activation and the part ribbon. Neither test checks the scenario's template chooser, pointer and keyboard interactions, or browser presentation. The native Rust scenario receipts provide evidence for those scenario-specific checks; they do not establish visual parity with Go.

On 2026-11-17, the five R4 creation scenarios in pinned source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` were reviewed against these
limited Go test premises. The feature SHA-256 was
`c6211904fc75d1a044949a77a5e1cd089522b4e8951d9ab2e66c2fa72da223a3`.
The two named Go tests passed again. They do not cover the template chooser,
mouse or keyboard input, or rendered result. These scenarios remain blocked
because this environment is not attested as the required capture host.
