# Ribbon panel visibility: visual observations

- Trigger: right-click a ribbon group and expand Show Panels.
- Layout: a light submenu lists panel names vertically beside the context menu. A checkmark column indicates visible panels; unchecked entries have no mark.
- Feedback: selecting an unchecked Parameters entry adds a labeled Parameters group to the dark ribbon. Reopening the list shows its checkmark. Selecting it again removes the group and the remaining panels reflow.
- Colors: light menu, dark ribbon, light command labels and blue checkmark accents as observed; these are not prescribed values.
- [Original schematic](panel-visibility-wireframe.svg) depicts the relationship between a visibility menu and a ribbon panel using generic geometry.

## Retain panel visibility while switching

- Trigger: show Parameters, switch from 3D Model to View and back, then switch between two open parts.
- Feedback: the Parameters group remains in the modeling ribbon when returning from View and appears in both observed part documents.
- Limit: same-session observation for two parts; restart behavior was not checked.

## View tab panel list

- Context: expanded View ribbon in a part.
- Feedback: Show Panels displays the four View groups as checked entries. Turning off Navigate removes the rightmost group; switching away and back leaves it absent. Its unchecked entry remains available to restore it.
- Limit: only the described same-session sequence was observed.

## Hide a compact group

- Context: the 3D Model ribbon uses compact group buttons and Parameters is visible.
- Feedback: right-clicking the compact ribbon exposes Show Panels. Turning off the checked Parameters entry removes its button and the neighboring buttons close the gap. The panel remains absent when the ribbon is expanded again.
- Limit: the sequence was observed for one panel in one session.

## Side-docked panel visibility

- Context: the part ribbon is docked left with Modify visible.
- Feedback: clearing Modify in Show Panels removes its group from the vertical ribbon, and other groups reflow into the space. The unchecked entry remains available in the same menu and restores the group when selected.
- Limit: observed for one panel and side dock in one session.

## Hidden panel across docking positions

- Trigger: hide Modify on the top ribbon, then dock the ribbon on the left.
- Feedback: the vertical ribbon omits Modify and its Show Panels entry remains unchecked. Selecting the entry restores the group.
- Limit: one panel and one docking change in the same session.

## Visibility in the floating ribbon

- Trigger: hide Modify on the top ribbon and undock it.
- Feedback: the floating vertical ribbon omits Modify; its context menu lists Modify without a checkmark. Selecting it inserts a labeled group between Create and Explore. Redocking at the top shows the restored panel.
- Limit: observed for one group within one session.

## Hidden floating group after redocking

- Trigger: hide Modify in the floating ribbon and dock it at the top.
- Feedback: Modify is absent from the top ribbon and unchecked in Show Panels. Selecting it restores the panel.
- Limit: one group and one redocking direction were observed within one session.

## Multiple panel choices

- Trigger: clear Modify and Pattern successively in Show Panels.
- Feedback: both groups disappear and both menu entries lose their checkmarks. Restoring Modify returns that group while Pattern stays absent and unchecked; restoring Pattern returns the second group.
- Limit: two panels on one tab were observed within one session.

## Interaction sequence wireframe

[Original step-by-step schematic](panel-visibility-steps.svg) shows the observed right-click, submenu, hidden state and restoration. It uses generic geometry and illustrative colors; it is not a captured application image.
