# Go rectangle baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/sketch_tools_test.go` checks four lines and one closed profile after two
synthetic clicks. `model/sketch/recipe_rectangle_test.go` checks four degrees
of freedom for a plain rectangle and positive dimensions on inverted input.
The application tests also cover prompt availability and a parameter-driven
locked dimension.

On Linux, the focused Go application and sketch tests passed using
`go test ./app -run '^(TestRectangleToolDrawsClosedProfileFromClicks|TestLineAndRectanglePromptsAndCancel|TestParameterDrivesARectanglesLockedDimension)$' -count=1`
and `go test ./model/sketch -run '^(TestRectangleRecipeDOF|TestRectangleRecipeHandlesInvertedDrag|TestRectangleYieldsOneClosedProfile)$' -count=1`.
These tests do not establish native-window layout, all typed-field interactions,
height-edit cancellation, or the Rust visual result. The nine historical Rust
native receipts checked those recorded scenario steps but are stale after native
test inputs changed. They do not establish the remaining rectangle scenarios or
an extrusion.

The Go command `go test ./app -run '^(TestFullModelingFlowViaUI|TestParameterDrivesARectanglesLockedDimension)$' -count=1 -v` also passed. The modeling-flow test uses synthetic session clicks and a stub profile picker, then checks one solid, six faces and its extrusion height. It is not a native-window test, a parameter-driven cube test, or a save/reopen and named-face test.
