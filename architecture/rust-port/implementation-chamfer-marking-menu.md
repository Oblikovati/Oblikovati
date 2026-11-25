# Part marking menu

The default Part profile follows the published eight-position layout. Measure,
Undo, and Hole route to the same actions as their ribbon commands. Revolve
appears disabled pending the modeling command. Customize searches the active
profile's command names and stages assignments to a radial slot or overflow.
The editor loads saved profiles when available and applies edits through the
marking-menu preferences owner (#3856).

The default Part marking menu offers Chamfer in its overflow. The selected-face
preset still has a Chamfer slot. Both start the existing tool with its normal
availability checks. A saved layout can also assign Chamfer to a radial slot.
A native X11 test opens Chamfer from the menu, cancels, and checks that the
feature list is unchanged (#4046).

Issues #4048 and #4050 track the outstanding selected-face commands.
