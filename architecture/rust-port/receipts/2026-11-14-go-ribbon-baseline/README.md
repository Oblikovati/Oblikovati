# Go ribbon baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`head/ui/chrome_ribbon.go` renders a fixed ribbon band at the top; it does not
implement the side docking described by this scenario.

On Linux with Xvfb, `go test ./ui -run '^(TestPanelsThatFitEverythingFits|TestPanelsThatFitCollapsesFromTheRight|TestRibbonRowWidthCountsDividersOnce)$' -count=1` passed from the `head` module. These tests check ribbon panel fitting and width calculation, not docking or the rendered side layouts. The pinned Rust native scenario checks the docking states and return to the top; its final capture shows the restored top layout.
