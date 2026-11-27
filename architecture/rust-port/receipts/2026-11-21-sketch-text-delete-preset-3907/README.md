# Delete a preset and keep accepted Sketch Text

All ten steps of pinned `text-39` passed on Linux with native X11 keyboard
and mouse input, Xvfb, Vulkan Lavapipe, and DejaVu Sans. The feature
SHA-256 was `0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`
at source revision `b576445b98cffd686dcc05aa8c017de2a591c8c8`.

The test created a native Part. It placed two blue stacked fractions at
different anchors and a separate label. It saved and reopened the Part
in the same desktop session, then opened the second fraction from its
visible outline. The second editor showed preset `g` before deletion.

Delete Current removed `g` immediately. The preset list offered No
Preset and Last Used, and the editor kept the fraction preview. Cancel
kept the complete saved sketch: both fractions, their identities and
anchors, and the separate label. The status bar showed Ready and 47
dimensions needed.

- [Preset list after deletion](preset-g-removed.png)
- [Accepted text after Cancel](accepted-text-after-delete.png)

The build used API v0.163.0 at
`c051de97568c0345921f0f900982da31eb9ddca4` and Martensite at
`bef25f47548431c8f128f08f66a1f719bcee4982`. This is a local baseline
record. Strict verification needs bounded build and runtime inputs.
[Issue #3978](https://github.com/Oblikovati/Oblikovati/issues/3978)
