# Viewport Orientation evidence baseline

The baseline covers all 20 Viewport Orientation declarations in scenario
snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`. It records the expected
results, Go reference and test limits, Rust test paths, and the locked API
and Martensite revisions for the current Rust baseline.

Go reference `aa8297d1b0ac33c024f19e54821536d76e8a447a` passed the following
focused checks on Linux. These checks cover Home, compass preferences,
ViewCube visibility and selection-lock preferences, projection enum
round-trip, camera snaps, roll, adjacent faces, and arrow animation.
The pinned scenario steps specify the required rendered state.

```sh
# Run from the Go repository.
go test ./app -run '^(TestHomeViewIsIsometricAndFramesModel|TestGoHomeWithoutCustomHomeUsesIso|TestShowCompassPersistsAndLoads|TestShowCompassNoStoreIsInSession|TestViewCubeTogglesPersist|TestViewProjectionRoundTripsOverTheWireEnum)$' -count=1
cd head
xvfb-run -a go test ./ui -run '^(TestHitTestCenterIsFrontFace|TestHitTestCornerZone|TestSnapCameraPlacesEyeOnRegionSide|TestVisibleFacesFromTopShowsTop|TestRolledViewRotatesUpKeepingFraming|TestAdjacentViewSnapsToNeighbourFace|TestApplyViewCubeArrowAnimates)$' -count=1
```

See [the desktop run](../2026-11-20-orientation-transition-3961/README.md) for
the 23 passing cases on the updated live feature. Strict acceptance requires
a current pin and verified build and runtime inputs under #3978.
