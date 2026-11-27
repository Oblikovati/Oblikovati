# Compact Ribbon Docking

The published feature passed 17 of 22 scenarios on application revision
`f518714080d0c4bbca8fe92640e0b275c5c64bf2`, API v0.165.0, and Martensite v0.13.1.
The [Cucumber report](cucumber.json) and [JUnit report](junit.xml) record
171 passed steps and 5 failed steps. The clean
[local snapshot](snapshot.json) captured the published feature, support
files, and three native Part fixtures. The feature SHA-256 was
`3314b980c97be4aa018b9628898ad8758179b0737ed40d4f71f75a31395bf026`. The scenario source remained unchanged during this run.

The two newly passing cases used generated native Parts with visible solids.
The floating Modify pane closed on an X11 viewport click. The same click
selected one face and displayed its highlight and the one-face status.
A click in clear viewport space removed both. The left-docked run replaced
Work Features with Modify, checked the blue group highlight and its
command contents, then closed the pane with an X11 click. Slot, Draft,
and Thread are displayed as unavailable. The command pane uses the
existing ribbon command grid.

All 46 GUI ribbon interaction tests and the
[separate X11 dock and restore test](title-x11.log) passed on the compact
group change before the Finish control update. The native
run used X11 with Xvfb and Vulkan Lavapipe. The [native log](native.log.gz)
records the first failure in each remaining case. The pinned application binding check reports 1,355 unresolved step
entries, down from 1,376. The full gate remains open under
[#3860](https://github.com/Oblikovati/Oblikovati/issues/3860).

The source update replaced the Inventor inputs for Shell and Finish Sketch
with native Parts. The [Finish Sketch scenario](sketch-final.log) also
passed in isolation. Finish is directly available in Exit, returns to
the 3D Model groups, and leaves the geometry and active Part unchanged.

Three cases still cover scrolling and the floating Annotate tab. The
Shell case needs its new fixture and preview steps bound. The
[separate Shell test](shell-refusal.log) still reproduces a tolerance
refusal on its existing test Part. The mixed document case requires
Assembly and a native Sheet Metal fixture.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
