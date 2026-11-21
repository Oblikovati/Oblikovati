# Go model browser baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/browser.go` builds the browser tree and `head/ui/browser_view.go` draws it.

On Linux, `go test ./app -run '^(TestBrowserNestsConsumedSketchUnderFeature|TestBrowserOrdersTimelineByCreation|TestBuildBrowserMakesSketchFeatureBodySelectable|TestSelectBrowserNodeIgnoresNonSelectable)$' -count=1` passed. These tests check tree placement, ordering, selectable nodes, and rejection of a nonselectable node. They do not check the rendered selection highlight, contextual toolbar, or clearing by clicking the viewport. The pinned Rust native scenario checks those interactions; its final capture shows the cleared selection, not the intermediate highlight.
