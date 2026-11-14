# Ribbon docking

In an active part workspace, right-clicking a ribbon group opens a context menu with Ribbon Appearance, Show Panels, Customize User Commands, Undock Ribbon, and Docking Positions. Docking Positions offers Top, Left and Right and marks the current position with a checkmark.

Choosing Left changes the horizontal ribbon into a vertical tab strip with stacked command-group buttons along the left edge. The left-docked model browser shifts inward, alongside the ribbon. Choosing Right moves the ribbon to the right edge, with vertical tabs and stacked command groups; the model browser remains on the left. Choosing Top restores the horizontal ribbon above the viewport. The same part remained active during this sequence.

These observations cover one part and one session. Persistence across restarts was not established. See [ribbon-docking.feature](ribbon-docking.feature) for review-tagged Gherkin scenarios and [VISUAL.md](VISUAL.md) for visual observations.

Choosing Undock Ribbon from the top-docked ribbon context menu opened a floating vertical ribbon over the workspace. Its 3D Model Create group opened a temporary adjacent command panel; clicking the viewport dismissed the panel while leaving the ribbon floating. In the floating ribbon context menu, Undock Ribbon was unavailable and choosing Top from Docking Positions restored the top dock.

With the 3D Model ribbon docked on the left, selecting Create opened an adjacent command list. Selecting Modify replaced it with the Modify list. Clicking the viewport dismissed the list while the ribbon remained docked. No command was executed.

In the observed floating 3D Model ribbon, scrolling the group area brought the partially visible Work Features group fully into view. Selecting it opened an adjacent panel with Plane, Axis and Point commands; clicking the viewport closed that panel and left the ribbon floating. The ribbon was returned to the top dock.

In a floating ribbon for an active part, selecting Annotate replaced the 3D Model group buttons with annotation groups. The Notes button opened an adjacent command panel. Selecting 3D Model closed that panel and restored the modeling groups, while the ribbon remained floating.

In one session with a part, an assembly and a sheet-metal part open, switching from the part to the assembly and then to the sheet-metal part kept the ribbon docked on the left. Its tabs and commands changed to suit the active document. This does not establish persistence after restart.
