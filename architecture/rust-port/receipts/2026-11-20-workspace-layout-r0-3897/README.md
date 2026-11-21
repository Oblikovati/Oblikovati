# Workspace Layout evidence baseline

The baseline covers the 44 Workspace Layout declarations in scenario snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. It records the expected results
from the pinned feature, the Go reference and test limits, the Rust entry
points, and the application dependency revisions. The changes record an R0
baseline. They do not mark a native scenario verified.

Go reference `aa8297d1b0ac33c024f19e54821536d76e8a447a` passed the following
focused checks. The Part tests cover the tree hierarchy, order and
selectability. The Drawing test covers browser nodes and their actions. The
head tests cover registry membership and open and close state. The captured
workspace geometry, search, dock interactions and View menu require native
Rust checks.

```sh
# Run from the Go repository.
go test ./app -run '^(TestBrowserNestsConsumedSketchUnderFeature|TestBrowserOrdersTimelineByCreation|TestBuildBrowserMakesSketchFeatureBodySelectable|TestSelectBrowserNodeIgnoresNonSelectable|TestDrawingViewBrowserNodesAndMenu)$' -count=1
cd head
xvfb-run -a go test ./ui -run '^(TestDockablePanelRegistryMembership|TestDockablePanelOpenCloseRoundTrips)$' -count=1
```

The pinned feature uses older fixtures and wording. In particular, the
pinned View menu names iLogic controls. The live feature names Rule Log and
Rule Automation. Strict acceptance needs a current pinned feature and
verified build and runtime inputs under #3978.
