# Part marking menu: Chamfer

The default Part marking menu's Chamfer slot starts the existing Chamfer tool.
It uses the tool's normal availability checks and honors configured menu slots.
A native X11 test opens Chamfer from the menu, cancels, and checks that the
feature list is unchanged (#4046).

This does not complete the selected-face context-menu scenario. Its separate
standard menu and other observed commands remain open (#4048, #4050).
