# Active document tab during native resize

The amended Document Tabs feature passed 40 of 41 scenarios on Wayland,
with 347 steps in passing cases. The four-step resize scenario also
passed on X11. The saved-tab preview scenario remains pending at the
mapped-or-network-path setup step.

The resize scenario opens the published solid Part alongside four more
Parts, then selects the saved Part through Documents. After the native
window is dragged down to about 660 × 550, the saved Part keeps its active
underline and both ends of its middle-truncated label. The right arrow
scrolls it out of view, the left arrow brings it back, and Documents
still opens while the saved Part stays active. The Wayland capture was
visually checked at the narrow size.

The resize tests used commit `5c15be9fa` and existing local GUI
filesystem edits. The preceding 40 Wayland cases used the same product
code and an earlier version of the new resize assertion. Wayland used
nested Weston 13 and X11 used Openbox, with Xvfb and Vulkan Lavapipe on
both. The published scenario, fixture, and support hashes matched before
and after the runs. Issue #3978 tracks strict build and runtime provenance.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 resize report](x11.json) and [step arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)
