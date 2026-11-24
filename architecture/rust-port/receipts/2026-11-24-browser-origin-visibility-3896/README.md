# Origin visibility in an empty Part and a solid Part

The [locked-view reproduction](before/locked-toggle/cucumber.json) failed
while choosing a viewport click. The helper requested scene-fit bounds,
which require geometry. It now reads the document viewport rectangle
directly and still excludes GUI windows and solid face hits.

| Pinned scenario | Passed steps | Receipt |
| --- | ---: | --- |
| Toggle an origin plane while the active view is locked | 12 | [Run](after/locked-toggle/cucumber.json) |
| Switch locked views after showing an origin plane | 10 | [Run](after/locked-switch/cucumber.json) |
| Show and hide an origin axis from its browser menu | 11 | [Run](after/axis/cucumber.json) |

The locked-view runs used `ee5ec8fbe` with local test changes. The axis
recheck used the original build at that revision. The
[solid X11 regression](x11-solid-outline.log) also passed with the revised
helper. The builds included existing local file-browser edits. The
source was `b576445b98cffd686dcc05aa8c017de2a591c8c8`, and the runs used
Linux X11, Xvfb, Openbox, and Vulkan Lavapipe.

All three pinned scenarios gained R0 records.
See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
