# Offset Plane properties at the minimum window size

At 640 × 480, the fixed-height panel left more than 100 pixels below the
controls. Offset Plane now uses the shared left label column, outlined
numeric field, panel border and shadow, and 72 × 28 action buttons. The
panel height follows the controls and wraps validation text inside its
bounds. [Before](before.png) · [After](after.png)

Five native Offset Plane tests passed. The narrow-window test entered
0 mm, checked the disabled OK, validation message and visible footer,
corrected the value, and clicked Cancel. No feature was created. Other
tests covered accepting an origin, work-plane or face reference, plus
undo, redo and reopening. Three GUI draft tests also passed.

The published Ribbon Customization scenario verified opening the grouped
Plane command and dismissing it with Escape without creating a feature
(11 steps). The pinned Projection Fixtures scenario accepted 50 mm and
started a sketch on the resulting work plane (4 steps).
[Grouped command](grouped-plane.json) · [Published input hashes](current-inputs.sha256)
· [Outside work plane](outside-plane.json) · [Pinned source](projection-source.json)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`606df6d8b` with the property layout change and pending local filesystem
and Ribbon Customization edits.
