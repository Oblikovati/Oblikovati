# Browser Find and dock controls on Wayland

Ten Find dialog scenarios passed 127 steps on Wayland. They cover feature and
sketch criteria, grouping and declining grouping, finding and clearing an
extrusion, no-match feedback, pending edits, and canceling saved-search
dialogs. Three Workspace Layout scenarios passed another 21 steps. They
cover the Model menu, the drawing browser menu, and search focus from the
inactive Model tab.
[Find report](find.json) · [Dock report](dock.json) · [Input hashes](inputs.sha256)

The Find setup originally tried to hover the dock title with X11 input on a
Wayland session. The standalone tests also looked for the old floating
Menu caption. They now locate the visible dock title control. The shared
dock driver dispatches hover, click, and text through the active backend.
The cross-tab scenario verifies that typing goes directly into the focused
Model search field and that reopening it from Log retains the query.
[Find baseline](find-baseline.json) · [Cross-tab typing baseline](dock-search-baseline.json)

Five native Find tests passed on Wayland. The same five and the native
dock-title check passed on X11. The published cross-tab scenario also
passed all 11 steps on X11. [X11 cross-tab report](dock-search-x11.xml)

The build used `4b15e1df0` with the test changes and existing local GUI
filesystem edits. Wayland used headless Weston with Pixman and scripted
window input. X11 used Xvfb, Openbox, and XTest for the native controls.
Both used Vulkan Lavapipe. The recorded step arguments match the published
scenarios. Issue #3978 tracks strict build and runtime provenance.
