<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Go point cancellation baseline

The Go checkout was at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
The following command passed on Linux:

```sh
go test ./app -run '^(TestSketchToolsLifecycleAndPrompts|TestPointAndSplineFinalPrompts|TestCenterPointModeMarksNewPoints)$' -count=1
```

`TestSketchToolsLifecycleAndPrompts` starts and cancels each sketch tool,
including Point, and checks its initial prompt and commit readiness. It does
not assert the state after cancellation. `TestPointAndSplineFinalPrompts`
checks a Point prompt after a click. `TestCenterPointModeMarksNewPoints` checks
the center role of a created point. None checks one Escape, the context menu,
the Ready status, or marker pixels. `app/sketch_geometry_tools.go` defines
Point cancellation as resetting its collected input; this is not evidence of
the complete desktop scenario.

The historical Rust point-11 receipt tests the pinned scenario with frame-loop
input. It is not current acceptance proof: build provenance is unbounded (#3978).
