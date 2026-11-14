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
