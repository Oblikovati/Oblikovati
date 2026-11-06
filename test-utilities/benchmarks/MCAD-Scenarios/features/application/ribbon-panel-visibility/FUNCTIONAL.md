# Ribbon panel visibility

With a part active and the 3D Model ribbon expanded, right-clicking a ribbon group opens a context menu. Show Panels expands a list of panel names with checkmarks beside visible panels. In the observed configuration, Parameters was unchecked, while Create and Modify were checked.

Selecting Parameters added a Parameters panel to the ribbon and caused its entry to appear checked on reopening the list. Selecting it again removed the panel. The same part remained active. The original visibility was restored after the observation.

The initial sequence was observed for one panel on one tab. Further same-session observations are described below; restart persistence and effects across document types were not established. See [ribbon-panel-visibility.feature](ribbon-panel-visibility.feature) and [VISUAL.md](VISUAL.md).

After enabling Parameters in one part, switching to View and back to 3D Model left that panel visible. Activating a second open part showed Parameters on its 3D Model ribbon, and returning to the first part retained it. This sequence was observed within one session; restart persistence was not checked. The panel was hidden again afterward.

On the View tab of the observed part, Show Panels listed Visibility, Appearance, Windows and Navigate, all checked initially. Selecting Navigate hid its ribbon group. Switching to 3D Model and back left Navigate hidden and unchecked in the View panel list. Selecting it again restored the group. This covers one tab change in the same session.

With Parameters visible and 3D Model reduced to compact group buttons, the ribbon context menu still offered Show Panels and marked Parameters checked. Selecting it removed the Parameters button from the compact row. Cycling back to the expanded ribbon left the panel absent. This restored the initial visibility and expanded display.

With the 3D Model ribbon docked left, its context menu offered Show Panels. Clearing the checked Modify entry removed that group and the remaining groups reflowed vertically. Reopening the menu showed Modify unchecked; selecting it restored the group. The ribbon was returned to the top dock.

Hiding Modify on the top-docked 3D Model ribbon and moving the ribbon to the left kept Modify hidden. Show Panels still listed it unchecked; selecting it restored the group. The ribbon was returned to the top. This sequence was observed within one session.

After hiding Modify on the top-docked ribbon, undocking produced a floating ribbon without that group. Its Show Panels menu listed Modify unchecked. Selecting it restored the group in the floating ribbon; docking at the top retained the restored panel. This was observed in one session.

Hiding Modify in the floating ribbon and docking it at the top left the panel hidden. Show Panels listed Modify unchecked, and selecting it restored the panel. After restoration, switching to View and back returned the ribbon to its usual height. The sequence was observed in one session.

Modify and Pattern could both be hidden through successive Show Panels selections. Both entries were unchecked on reopening. Restoring Modify left Pattern hidden and unchecked; selecting Pattern then restored both panels. This was observed on the 3D Model ribbon within one session.

A cross-tab sequence hid Modify on 3D Model and Navigate on View. Returning to 3D Model showed Modify still hidden. Restoring Modify and returning to View left Navigate hidden and unchecked; it was then restored. This observes separate choices for these two panels in one session.

With Show Panels expanded on the 3D Model ribbon, pressing Escape closed the panel submenu while leaving the parent context menu open. A second Escape closed the context menu. The observed visible and hidden panels did not change.
