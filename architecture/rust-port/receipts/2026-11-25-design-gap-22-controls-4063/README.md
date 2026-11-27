# ViewCube hotspots and controls

The standard cube face is 64 px square. The navigation bar shares its
horizontal center in all four positions. The viewport remains visible
around the cube and through its contact shadow. The Home outline,
cardinal arrows, mirrored roll arcs, and menu chevron fade in during
hover and fade out when the pointer leaves. The cube retains its
configured inactive opacity.

The same projected regions drive hover feedback and picking. A face-on
pick keeps the existing face and corner commands. Oblique edge and
corner picks use their model-space normals and preserve the camera's
target and scale.

Two focused native X11 checks passed on Linux with Openbox and Vulkan
Lavapipe. Their captures show the idle cube, corner highlight, options
menu, and the Front label after a quarter turn.
[Idle](viewcube-idle.png) · [Corner](viewcube-corner-hover.png) ·
[Menu](viewcube-chevron-menu.png) · [Rotated](orientation-native-rotated-3961.png)

The GUI geometry and host camera tests passed. The orientation checks
sample unhighlighted face paint when looking for diagonal seams and use
the rendered radio marks to verify scope selection.
[Scope scenario](scope/cucumber.json)

The complete published feature passed all 23 scenarios and 253 steps
on the native X11 desktop.
[Full Cucumber report](full/cucumber.json) · [JUnit](full/junit.xml)
