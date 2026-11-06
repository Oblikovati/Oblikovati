# Ribbon display modes

The ribbon presents commands in named groups beneath a row of tabs. With an active part and View selected, the expanded ribbon shows its groups and commands. A display control can reduce the space used by the ribbon. With Cycle Through All selected, successive activations in the observed sequence changed the expanded ribbon to compact group buttons, then a narrow row of group titles, then tabs only. The viewport gained vertical space as the permanent ribbon became smaller.

Selecting a compact group button or title opens a temporary panel containing that group's commands. In tabs-only mode, selecting a tab temporarily reveals its commands; clicking outside closes the temporary panel while the tab remains selected. Continuing the observed cycle restored an expanded ribbon that stayed visible after clicking the viewport.

The display-control menu offers Minimize to Tabs, Minimize to Panel Titles, Minimize to Panel Buttons, and Cycle Through All. Choosing Minimize to Tabs directly hid the permanent groups, and the menu subsequently showed that choice checked. The cycle setting was restored after the observation.

These observations cover a part workspace and the described View, Windows and 3D Model interactions. They do not establish restart persistence, keyboard access, or every direct minimization choice. See [ribbon-display.feature](ribbon-display.feature) and [VISUAL.md](VISUAL.md).

In an observed drawing workspace, double-clicking the active Place Views tab advanced the ribbon through full commands, compact group buttons, group titles, tabs only, and back to full commands. The display control was configured to cycle through all modes. The drawing remained active throughout.

In one same-session switch from a drawing to a part and back, the compact group-button display remained in effect while the groups changed to those appropriate to the active document. Returning to the drawing restored its compact group buttons.

In compact group-button mode on a drawing, selecting Create opened its temporary drawing-view commands. Selecting Modify replaced that panel with the Modify commands. Clicking the drawing outside the panel dismissed it and left the compact buttons visible.

With the drawing ribbon in compact group-button mode and the Place Views Create panel open, selecting Annotate closed the temporary panel and replaced the compact buttons with the Annotate tab's groups.

In the drawing workspace, choosing Minimize to Panel Titles directly reduced the active ribbon to its group-title strip. With that choice selected, successive activations of the display control alternated between the full ribbon and the title strip; the menu marked Minimize to Panel Titles as selected. Cycle Through All and the expanded ribbon were restored afterward.

Choosing Minimize to Panel Buttons directly in the drawing workspace showed compact group buttons. With that choice selected, the display control alternated between the full ribbon and compact buttons, and the menu marked the choice as selected. Cycle Through All and the full ribbon were restored afterward.
