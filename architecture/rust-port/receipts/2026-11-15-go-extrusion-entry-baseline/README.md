# Go extrusion entry baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/commands_standard.go` registers Extrude and `app/extrude_tool.go` defines
its profile selection. `app/create_sketch_tool_test.go` exercises selecting a
sketch plane in the viewport and the browser.

On Linux, this command passed:

```sh
go test ./app -run '^(TestCreateSketchToolFlow|TestCreateSketchToolPicksViaBrowser|TestExtrudeDisabledInsideSketch)$' -count=1
```

These tests cover plane selection and the Extrude availability gate. They do
not establish the complete Extrude-to-sketch journey or native input delivery.
