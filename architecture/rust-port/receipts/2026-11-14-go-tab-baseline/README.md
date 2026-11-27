# Go document-tab baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`head/ui/chrome_doctabs.go` activates the selected document through the
workspace and follows active-document changes when displaying its view.

On Linux, `go test ./app -run '^TestPerDocumentStreamsAreIsolated$' -count=1`
and, from `head`, `xvfb-run -a go test ./ui -run '^TestDocumentTabLabelDisambiguatesCollidingNames$' -count=1`
passed. The first checks isolated history after programmatic activation; the
second checks unique tab labels for colliding names. Neither test verifies a
mouse click, window title, browser contents, or rendered sketch edges. Those
need separate native Rust scenario evidence.

`go test ./app -run '^TestCloseBackgroundDocumentKeepsActiveEditState$' -count=1 -v` passed. It closes a background document through the session and checks that the active sketch and tool remain. It does not test the tab close button, tab context menu, or saved-file bytes.

`go test ./app -run '^TestCloseDocumentLeavesSketchEnvironmentAndDisarmsTool$' -count=1` and, from `head`, `xvfb-run -a go test ./ui -run '^TestDocumentCloseGuardRequestsPromptOnlyForDirtyDocument$' -count=1` passed. They cover session cleanup after close and the dirty-document prompt guard. They do not test the native tab-cancellation journeys or the scenario's restored viewport.

On 2026-11-17, the eight R4 tab scenarios in pinned source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` were reviewed against these
limited test premises and the save/open baseline. The feature SHA-256 was
`2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f`.
The five named app tests and the two named UI tests above passed again on
Go `aa8297d1b`. The tests do not establish the complete scenario journeys;
the scenario classifications remain unverified. The recorded Rust oracle
describes required checks, not a new acceptance result.
