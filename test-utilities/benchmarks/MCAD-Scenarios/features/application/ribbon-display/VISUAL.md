# Ribbon display: visual observations

## Expanded View tab

- Context: an open part document with the View tab selected.
- Layout: labeled tabs form a row above a wide command region. Controls are arranged in named groups, mixing larger icons with smaller buttons and menus. Group names appear along the bottom of the region. The model browser and viewport begin below the ribbon.
- Feedback: the selected tab has a contrasting background. A small control near the tabs changes the ribbon presentation.
- Color: the observed theme uses dark blue-grey surfaces, light text and blue highlights. These colors are local observations, not mandatory values.

## Compact groups and title strip

- Trigger: activate the ribbon display control from the fully expanded state, then activate it again.
- Layout: the first compact state substitutes one icon-and-label button for each group. The next state reduces the groups to a narrow row of titles. The viewport gains height in each reduction.
- Feedback: selecting the Windows group button or title displays its controls in a temporary panel below the chosen group.

## Tabs-only and return to expanded ribbon

- Trigger: advance once more from the group-title strip. Select the View tab to open its commands, advance the display control, then click empty viewport space.
- Layout and feedback: tabs remain visible with no permanent group strip. Selecting a tab opens its commands over the upper workspace. After the subsequent display-control action and outside click, the expanded command region remains visible.
- Original icon brief: use a fresh stack of horizontal bands with a small directional marker to convey cycling workspace density; preserve a text or accessible label.
- Limits: exact animation, keyboard access, persistence and behavior for every tab were not established. The order above describes the observed sequence only.

## Display control menu

- Trigger: open the small menu beside the ribbon display control.
- Feedback: a compact vertical menu lists minimization to tabs, panel titles and panel buttons, plus a cycle-through-all option. In the observed state, cycle-through-all is checked. The preceding cycling sequence is conditional on that setting.

## Choosing a specific minimization mode

- Trigger: choose Minimize to Tabs directly in the display control menu, select a different ribbon tab, then click outside its temporary panel.
- Feedback: the command area collapses immediately. The chosen tab opens its own commands temporarily, which close on an outside viewport click while the selected tab remains marked. Reopening the display menu shows a checkmark beside Minimize to Tabs. The prior cycle-through-all option was subsequently restored.
- Limits: preference persistence after restarting the application remains unobserved. The other direct minimization choices are described below.

## Double-click the active drawing ribbon tab

- Context: a drawing is active and the ribbon display control is configured to cycle through all modes.
- Trigger: repeatedly double-click the active Place Views tab.
- Feedback: the full command panel changes in order to compact Create, Modify, Sketch and Sheets group buttons, then a narrow group-title strip, then tabs only, and finally the full command panel. The drawing remains active and the sheet occupies more vertical space as the ribbon collapses.
- Limits: the sequence was observed for one drawing tab and one display-control configuration; no command was executed.

## Switch document types with compact group buttons

- Context: a drawing and a part are open; the drawing ribbon is in compact group-button mode.
- Trigger: activate the part tab, then return to the drawing tab.
- Feedback: the part ribbon displays its own compact group buttons, such as Sketch, Create and Modify. Returning to the drawing displays its Create, Modify, Sketch and Sheets group buttons. The compact display stays active throughout.
- Limits: this covers one switch in one session and does not establish persistence after restarting. The expanded ribbon was restored after observation.

## Switch temporary drawing ribbon groups

- Trigger: in compact group-button mode, select Create, select Modify, then click an empty area of the drawing.
- Feedback: a temporary Create panel opens directly beneath the compact buttons, showing drawing-view commands. Selecting Modify replaces it with a shorter panel of modification commands. The outside click closes the panel while the compact buttons remain.
- Limits: no drawing command was invoked; expanded ribbon display was restored.

## Switch tabs while a compact group panel is open

- Trigger: open the Place Views Create group in compact mode, then select Annotate.
- Feedback: the temporary Create panel closes and the compact button row changes to Annotate groups, including Dimension, Text and Symbols. The drawing sheet remains visible.
- Limit: no command was executed; the full ribbon was restored afterward.

## Original visual assets

- [Ribbon modes wireframe](ribbon-modes-wireframe.svg) schematically compares the four observed permanent ribbon densities; generic groups represent controls without reproducing source artwork.
- [Ribbon density icon](ribbon-density-icon.svg) is a newly drawn colored symbol based on the brief above.

## Choose panel-title minimization

- Trigger: open the display menu in an expanded drawing ribbon and choose Minimize to Panel Titles.
- Feedback: the command area becomes a narrow strip of Annotate group titles. Activating the display control expands the full command area; activating it again returns to the title strip. The menu then shows a checkmark beside Minimize to Panel Titles.
- Limits: this was observed in one drawing session; restart persistence was not checked. The original cycle setting and full ribbon were restored.

## Choose panel-button minimization

- Trigger: choose Minimize to Panel Buttons in the display menu of an expanded drawing ribbon.
- Feedback: Annotate groups become compact icon-and-label buttons. The display control alternates between the full command area and the compact buttons; reopening the menu shows a checkmark for the panel-button choice.
- Limits: one drawing and session were observed; restart persistence was not checked. The cycle setting and expanded ribbon were restored.

## Tooltip in tabs-only mode

- Context: a part workspace with only the ribbon tabs permanently visible.
- Feedback: with the pointer resting over the display control, a small light rectangular tooltip appears just below and to its right, labeled Show Full Ribbon. The dark viewport remains visible.
- Limit: the tooltip identifies the offered action; the activation outcome was not established in this observation.

## Toggle tabs-only and full display

- Context: an active part with Minimize to Tabs selected in the display menu.
- Trigger: activate the ribbon display control repeatedly.
- Feedback: the full 3D Model command area disappears, returns on the next activation, and disappears again on the next. The tab row and part viewport remain visible.
- Limit: observed in one part and session; restart persistence was not checked.
