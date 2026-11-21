# Native Point Text case conversion

The focused Linux X11 test passed under Xvfb with Vulkan Lavapipe. Native mouse
and keyboard input opened an existing Text object. UPPERCASE without a selection
left `ab` unchanged in the editor and the viewport. Ctrl+A followed by UPPERCASE
showed `AB` in both. Accepting the edit kept the Text identity and anchor.

The focused test built the binding change in a worktree based on `8c4d03d28`.
The published scenario then ran through the same executable. It passed its
first five steps and stopped at the undefined strike-through action. The
binding report also identifies the missing visual strike-through assertion.
The scenario stays blocked on the font metrics in
[Martensite #578](https://github.com/Oblikovati/Martensite/issues/578).

The published `.feature` SHA-256 matched before and after the replay.
The screenshot shows selected `AB` in Format Text and the converted sketch
glyphs below the editor.
