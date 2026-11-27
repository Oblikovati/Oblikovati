# Full pinned Ribbon Keyboard Navigation run

All 18 scenarios and 219 steps passed in the
[native run](cucumber.json). The [binding inventory](step-bindings.json)
reports zero unresolved steps. The scenarios cover ribbon tabs and commands,
display modes, compact groups, face selection, camera motion, and the
expanded model browser.

The Head build included the application changes in `e05f0d6c7` and
existing local edits to the document file-browser code. The run used X11,
Xvfb, Openbox, and Vulkan Lavapipe. The pinned scenario commit was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The scenario file SHA-256 was
`835ba9fd1dfa3e007e9555b4b890cbf75a006e40c4e14b4285709f93ab5889d4`.

The 18 pinned declarations gained R0 baseline records. The published
feature has a different source hash and was outside this run.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
