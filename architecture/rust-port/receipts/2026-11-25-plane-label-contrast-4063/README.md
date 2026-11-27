# Origin plane label contrast

The zoomed XY Plane label measured 1.74:1 in Dark against the pale model
face before this change. It now has a solid theme-colored backing when
the plane faces the viewer. The backing stays in the plane, shares the
label’s hit area, and fades with the text as the plane turns edge-on.
[Before](before.png) · [Dark](dark.png) · [Light](light.png)
· [Oblique view](oblique.png)

At 1000% zoom, the native test verifies at least 4.5:1 contrast between
the label ink and the sampled backing in Dark and Light, then uses the
label to start a sketch. XZ and YZ stay hidden in Front view. The four
origin picker checks, Extrude shortcut check, and six GUI tests passed.

Both pinned sketch scenarios passed, covering creation and cancellation
in nine steps. [Creation](pinned-0.json) · [Cancellation](pinned-1.json)
· [Pinned source](pinned-source.json)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`5c62dc301` with the label change and pending local filesystem and
Ribbon Customization edits.
