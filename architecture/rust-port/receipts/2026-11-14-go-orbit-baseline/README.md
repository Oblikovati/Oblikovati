# Viewport Orbit reference baseline

Go reference: `aa8297d1b` (`go-legacy`). Run from the Go `head` module:

```sh
xvfb-run -a go test ./ui -run '^(TestClassifyOrbitZone|TestApplyOrbitZones|TestApplyNavigationConstrainedOrbit|TestApplyNavigationModalFKeys)$' -count=1
```

Result: pass on Linux. These tests cover orbit-zone classification, camera
rotation, constrained orbit, and modal navigation. They do not establish the
navigation bar menu, guide rendering, context menu, or incremental-rotation
dialog required by the pinned scenarios. The four scenarios remain unverified
until their native runs and captures are reviewed.
