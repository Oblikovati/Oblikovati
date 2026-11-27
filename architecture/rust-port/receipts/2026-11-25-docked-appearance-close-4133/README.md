# Close and restore the docked Appearance Browser

The published scenario now clicks the Appearance tab's close control.
Its previous binding searched the browser contents for a Close caption.
The shared step locates the control in the rendered dock tab.

All 21 steps passed on Linux X11 with the published Part. Material
remained active when Appearance closed. View → User Interface restored
Appearance to the same dock and showed its contents with the Part visible.
[Scenario report](appearance/cucumber.json) ·
[Material after close](material-after-close.png) ·
[Appearance restored](appearance-restored.png)

The shared panel close and restore case passed all nine steps. The
separate X11 pointer test verified Material was active in the first
frame after Appearance closed.
[Shared panel case](panel/cucumber.json) ·
[Scenario and fixture hashes](current-inputs.sha256)

The build used revision `eb72dffbe` with the corrected step and pending
local filesystem and Ribbon Customization edits.
