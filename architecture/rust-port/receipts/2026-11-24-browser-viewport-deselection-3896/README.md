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

## X11 origin visibility round trips

The standalone X11 tests at `f1d5ac110` still used the fixed click. The
plane remained highlighted after hiding it, and the shown axis did not
return to its muted color. Both tests now use the unobstructed viewport
point and check that the row highlight clears after each toggle.

All six standalone origin-menu tests and strict desktop Clippy passed.
The pinned [plane visibility](x11-origin-plane/cucumber.json) and
[axis visibility](x11-origin-axis/cucumber.json) scenarios each passed
11 steps. The feature SHA-256 stayed at
`603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
