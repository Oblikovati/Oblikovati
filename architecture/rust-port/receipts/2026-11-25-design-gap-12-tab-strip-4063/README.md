# Document strip and window list

The Documents control shares the full 26 px tab strip and centers its button
vertically, leaving both border edges visible. The active tab underline sits
near the bottom of the strip to give the caption descenders clearance.

On X11 the native check measured the centered button and at least 5 px
between the active caption and the accent. It also opened the Documents list
from that button. [Tab strip capture](document-list-and-caption.png)

The published window list scenario passed all 12 steps. It activated a
Drawing from the menu and verified that the Part remained open.
[Window list scenario](window-list/cucumber.json)
