# Point cancellation native run

The focused run used application commit `6866c0268` and scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f`. It passed point-11 and all eight
steps under Xvfb and Vulkan Lavapipe. The capture tool records the command,
source and artifact hashes in the JSON receipt. The screenshot shows the sketch
and Ready status after the menu closes. The test checks unchanged sketch
geometry and history, and checks two framebuffer locations for absent markers.

The `full-feature` folder contains the earlier full Point run with the same test
code before its commit. It passed 3 of 9 scenarios and failed 6. The failed
cases include unbound steps and fixture or marker assertions. The captured
scenario source stayed clean and unchanged during that run.

These are historical results. The native test injects events into the window
frame loop, and runner build provenance is not bounded (#3978). They do not
establish OS input delivery or verified BDD status.
