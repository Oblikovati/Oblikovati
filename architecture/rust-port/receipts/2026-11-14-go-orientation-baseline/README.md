# Viewport orientation reference baseline

Go reference: `aa8297d1b` (`go-legacy`). The following commands passed on
Linux:

```sh
go test ./app -run '^(TestHomeViewIsIsometricAndFramesModel|TestGoHomeWithoutCustomHomeUsesIso)$' -count=1
cd head
xvfb-run -a go test ./ui -run '^(TestHitTestCenterIsFrontFace|TestHitTestCornerZone|TestSnapCameraPlacesEyeOnRegionSide|TestVisibleFacesFromTopShowsTop|TestRolledViewRotatesUpKeepingFraming|TestAdjacentViewSnapsToNeighbourFace)$' -count=1
```

The app tests cover the default isometric Home view. The head tests cover
face and corner hit testing, camera snap direction, visible Top, roll and
adjacent-face orientation. They do not establish the complete native UI
scenarios or their model-browser and visible-solid assertions. Those scenarios
remain unverified until their native runs and captures are reviewed.
