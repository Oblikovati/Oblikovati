# Ribbon docking: visual observations

- Top: a horizontal row of tabs sits above a wide region of grouped commands. The browser is below at the left and the viewport fills the remaining area.
- Left: vertical tab labels border a narrow column of stacked command-group buttons. The browser sits immediately to its right and the viewport fills the remaining space.
- Right: vertical tab labels and stacked group buttons occupy the right edge; the browser is on the left and the viewport remains between them.
- Menu feedback: a light context menu overlays the dark workspace. Its Docking Positions submenu lists the three positions and uses a checkmark for the active position.
- Colors: dark blue-grey workspace and panel surfaces, light text, blue selection accents. These are observations, not prescribed colors.
- [Original schematic](ribbon-docking-wireframe.svg) illustrates the three layouts with generic shapes and no source artwork.

## Floating ribbon

- Trigger: choose Undock Ribbon from the top-docked ribbon context menu.
- Layout: a narrow dark floating panel overlays the browser and viewport. It has a small title bar, vertical tab labels, stacked icon-and-label group buttons, and a vertical scrollbar.
- Feedback: selecting Create highlights its button and opens a dark command list alongside the floating ribbon. Clicking outside dismisses the list. In the context menu, Undock Ribbon appears disabled while floating; selecting Top under Docking Positions restores the top dock.
- Limit: movement and resizing of the floating panel were not established.

## Side-docked group panels

- Context: the 3D Model ribbon docked at the left.
- Feedback: Create highlights blue and opens a dark vertical command list to its right, over part of the browser. Selecting Modify moves the highlight and replaces the list with Modify commands; an outside viewport click dismisses it.
- Limit: this sequence covered the left dock in one part workspace.

## Scroll floating ribbon groups

- Trigger: scroll down over the floating 3D Model ribbon group column.
- Feedback: group buttons shift upward within the panel and the previously clipped Work Features button becomes fully visible. Selecting it highlights the group and shows a compact adjacent command list; an outside click closes that list.
- Limit: the exact scroll distance and behavior of other tabs were not established.

## Switch tabs in the floating ribbon

- Trigger: select Annotate, open Notes, then select 3D Model.
- Feedback: the vertical group column changes to annotation groups; Notes opens an adjacent command list. Switching back closes the list and replaces the column with modeling groups, without docking the panel.
- Limit: one part and one pair of tabs were observed.

## Switch document types with a side-docked ribbon

- Context: a part, assembly and sheet-metal part are open with the ribbon docked left.
- Feedback: activating the assembly keeps the vertical ribbon on the left while replacing its tabs and commands with assembly controls. Activating the sheet-metal part keeps that position and displays sheet-metal controls. The browser and viewport reflect the active document.
- Limit: observed for this sequence in one session.
