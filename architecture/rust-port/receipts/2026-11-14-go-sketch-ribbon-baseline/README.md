# Sketch ribbon reference baseline

Go reference: `aa8297d1b` (`go-legacy`). Run from the Go application root:

```sh
go test ./app -run '^(TestSketchTabPanelsMatchInventor|TestSketchTabButtonStyles|TestSketchTabIsContextual|TestVariantsEnabledOnlyInSketch)$' -count=1
```

Result: pass on Linux. `TestSketchTabIsContextual` asserts that the Sketch tab
is absent outside sketch edit and present inside it. The panel-alignment and
button-style tests inspect the in-sketch ribbon structure. The variants test
checks enabled rectangle variants inside a sketch; its name does not establish
what happens outside one. These tests do not inspect the native window, tooltip,
or all enabled states requested by the scenarios.

The pinned scenarios require the Sketch tab in an inactive Part sketch
context. This differs from the Go test. The Rust unit test
`ribbon::tests::document_tabs_keep_sketch_context_off_other_document_types`
passes and checks that a Part has the Sketch tab outside sketch edit. It does
not establish all command availability or the three native scenarios. Those
scenarios remain unverified.
