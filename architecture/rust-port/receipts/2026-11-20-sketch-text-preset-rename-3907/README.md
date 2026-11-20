# Rename and save a modified Sketch Text preset

Pinned `text-31` passed all six native steps. The X11 fixture created preset
`f` from `ab`, accepted the derived Text, and edited the Text to `c`. It saved,
closed, and reopened the native part in the same application session. Editing
the glyph then showed `*f`. Rename Current changed the caption to `*g`, and Save
Current changed it to `g`. The canvas and committed sketch stayed unchanged
through the rename and preset save.

- [Pinned Cucumber report](pinned-text-31/cucumber.json)
- [Modified preset after rename](renamed-dirty-preset.png)
- [Preset after Save Current](saved-renamed-preset.png)

The preset library and Text links last for the application session. The link
uses the document GUID, sketch ID, and Text ID. The file does not store the
session's preset library.

The focused Linux run used X11 under Xvfb, Vulkan Lavapipe, scenario snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`, and
`OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-31`.
Strict acceptance still requires bounded build and runtime inputs under #3978.

The unfiltered live Text run passed 11 of 38 scenarios. The other 27 stopped
at undefined steps. The feature and support hashes matched before and after
execution. [Cucumber report](cucumber.json)

All five native Text regression tests, the eight steps in pinned `text-24`,
all 12 GUI Text unit tests, and workspace Clippy passed.
