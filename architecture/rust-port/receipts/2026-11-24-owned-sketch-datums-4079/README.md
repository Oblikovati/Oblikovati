# Document-owned sketch inspection

Issue #4079. The native runs use behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Pinned scenario | Before | Current |
| --- | --- | --- |
| new-sketch-01 Create a sketch on a named origin plane | | [5 steps](pinned/origin/cucumber.json) |
| new-sketch-07 Create an empty sketch on a planar solid face | [6 steps](before/face/cucumber.json) | [6 steps](pinned/face/cucumber.json) |
| new-sketch-08 Create a sketch on a non-origin work plane | [4 steps](before/plane/cucumber.json) | [4 steps](pinned/plane/cucumber.json) |

Each native scenario verifies the sketch editing environment and the Fully
Constrained status. The two derived-support cases also finish the sketch
and check that the sketch and its support remain visible in the browser.

The owner regression starts with a 2 cm prism and a face sketch. A draft
depth of 4 cm moves the sketch plane to 4 cm. Reading the draft leaves
its generation and transaction intact and publishes no current replay.
Abort restores the 2 cm plane. A committed 3 cm depth moves it to 3 cm.
