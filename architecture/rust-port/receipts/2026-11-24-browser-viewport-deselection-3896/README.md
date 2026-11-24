# Browser selection clears on an empty viewport click

The [nested sketch](before/sketch/cucumber.json) and
[Find criteria](before/find/cucumber.json) cases both reached the final
deselection check with their highlights still visible. The
[failed frame](before/sketch/capture.png) shows the right-hand browser
covering the test's fixed click at (1300, 800).

The shared test step now picks a point inside the current document
viewport, outside GUI windows, with no solid face along the view ray.
The tests still drive the desktop with pointer events and check rendered
highlights and solid visibility.

| Pinned scenario | Passed steps | Receipt |
| --- | ---: | --- |
| Inspect a nested sketch from the model browser | 10 | [Run](after/sketch/cucumber.json) |
| Group search criteria for the same feature property | 16 | [Run](after/find/cucumber.json) |
| Show and hide an origin plane from its browser menu | 11 | [Run](after/plane/cucumber.json) |

The runs used X11, Xvfb, Openbox, and Vulkan Lavapipe. The reproductions
used the build at `d17002295`, and the passing runs used `712567169`
with the local test helper change. Both builds included existing
file-browser edits. The source commit was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The three pinned scenarios gained R0 baseline records.
See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
