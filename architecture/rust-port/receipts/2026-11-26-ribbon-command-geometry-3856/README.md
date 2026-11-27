# Ribbon command geometry on Wayland and X11

The amended split-button scenario passed all six steps on both backends.
The run inspected the six Part ribbons and checked the Work Plane and
Sketch dropdown markers against their command cells and nearby icons.
It checked full-height command artwork, left-aligned selector captions
in Tools and View, and the display policy menu at 640 px.

Each combined Ribbon Display report has 50 of 52 passing scenarios
and 569 steps in passing cases. The full runs used `5672c65cb`.
At 170% text size, the full runs stopped the narrow Tools scenario on
its exact-caption assertion. The two individual reruns used
`70b8fd5cc` and inspected the complete tooltip for each abbreviated
group. Each combined report includes that backend's corrected case.
The two remaining scenarios require a valid Shell preview.

The head library runner used `OBK_BDD_AREA=application`,
`OBK_BDD_FEATURE=ribbon-display`, and the shared scenario directory as
`OBK_BDD_ROOT`. Wayland used nested Weston 13 and X11 used Openbox.
Both ran in Xvfb with Vulkan Lavapipe. The test builds included existing
local GUI filesystem edits. The source, support, and fixture hashes
matched before and after the runs. Issue #3978 tracks strict provenance.

- [Wayland suite](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 suite](x11.json) and [step arguments](x11-arguments.json)
- Narrow Tools reruns on [Wayland](wayland-narrow-tools.json) and [X11](x11-narrow-tools.json)
- [Published input hashes](current-inputs.sha256)
