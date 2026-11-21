# Viewport Zoom reference baseline

Go reference: `aa8297d1b` (`go-legacy`). Run from the Go `head` module:

```sh
xvfb-run -a go test ./ui -run '^(TestApplyNavigationZoomsOnWheelWhenHovered|TestApplyNavigationWheelZoomsTowardCursor|TestApplyNavigationWheelZoomsTowardViewCentre|TestApplyNavigationModalFKeys)$' -count=1
```

Result: pass on Linux. The tests cover wheel zoom behavior and modal navigation.
They do not establish the navigation bar menu, Zoom All, window selection,
selected-face behavior, or cancellation required by these scenarios. The Rust
native tests check those behaviors against the pinned scenarios. The visible
solid and some starting camera states are prepared through the host.
