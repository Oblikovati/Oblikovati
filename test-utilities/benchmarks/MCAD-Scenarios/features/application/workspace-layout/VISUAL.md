# Workspace layout: visual observations

## Open part document with model browser

- Context: an open part document in the modeling environment. The model browser is visible and docked at the left.
- Layout: a narrow top strip contains application controls and document context. A horizontal ribbon of labeled tabs and grouped controls sits below it. The model browser occupies the left side; the large viewport fills the remainder. Orientation and navigation controls sit toward the upper-right and right edge of the viewport. A row of open document tabs and a status area run along the bottom.

```text
+--------------------------------------------------------+
| application controls / document context                |
| ribbon tabs                                            |
| ribbon groups and commands                             |
+-----------+--------------------------------------------+
| model     | viewport                         orientation|
| browser   |                                 navigation |
| tree      |                                            |
+-----------+--------------------------------------------+
| open document tabs                                     |
| status                                                 |
+--------------------------------------------------------+
```

- Feedback: the active ribbon tab determines which command groups are displayed. The User Interface menu presents independent checkable visibility options. Clearing Model Browser hides the left pane and allows the viewport to occupy its space; selecting it again restores the pane and checkmark.
- Observed color: dark blue-grey surfaces, pale text and blue accents. Checkmarks communicate enabled visibility; no exact palette requirement is inferred.
- Original icon brief: use a fresh, simple split-window motif for a panel visibility command; a filled narrow side region can denote the browser without copying a source icon.
- Limits: The hide-and-restore observation alone does not establish resizing or persistence across sessions. The wireframe is schematic rather than a pixel-accurate reconstruction.

## Floating and redocking the model browser

- Trigger: drag the browser header from the left dock into the viewport, then drag the floating header to the left edge.
- Layout and feedback: the browser becomes a compact rectangular floating panel that overlays the model, retaining its tree, header and scrollable content. The viewport occupies the former left-dock space. Returning the panel to the left edge restores the full-height docked browser and reduces the viewport width.
- Limits: Intermediate drag targets, sizing limits and persistence were not established by this sequence.


## Dock on either side

- Trigger: drag the browser header directly from the left edge to the right edge, and then back to the left edge.
- Layout and feedback: the browser occupies a vertical dock on the chosen side; the viewport and its orientation and navigation controls occupy the remaining area. The model tree remains present.
- Limits: top and bottom docking, size constraints and layout persistence were not established.


## Resize a docked browser

- Trigger: drag the vertical divider between the left-docked browser and viewport right, then back left.
- Feedback: the browser widens and the viewport narrows as the divider moves; reversing the drag restores their earlier widths. The tree remains available, and the model stays visible in the resized viewport.
- Limits: minimum and maximum widths, other dock positions and persistence were not established.
- Original icon brief: use a distinct paired-panel outline with a movable vertical divider and opposed arrows to suggest resizing.

## Browser tree menu

- The browser header menu is a compact vertical list offering Autoscroll, Expand All, Collapse All, Find, Display Preferences and Help.
- Collapse All hides child rows beneath their parent rows while leaving the model visible. Expand All reveals nested rows; in the observed long tree, the browser also displayed a lower scroll position afterward. The viewport continued to show the same model.
- Limits: no persistence or exact resulting scroll-position rule was established.
- Original icon brief: use distinct branching lines with small plus and minus badges for expand and collapse actions.

## Browser search

- Trigger: select the search control in the model browser header and enter a partial feature name.
- Feedback: a narrow search field appears beneath the header. In the observed search for work points, matching rows appeared under the document root, matching text was highlighted, and unrelated rows were filtered from the visible tree. The viewport stayed unchanged.
- The field's trailing close control first cleared a populated query and restored the unfiltered tree; using it again with the field empty hid the search field.
- Limits: case sensitivity and matching rules for other object types were not established.
- Original icon brief: use a fresh outlined lens and a distinct clear mark for searching and clearing browser content.

## No matching browser search

- An unmatched query replaces the tree rows with a small centered no-results message beneath the search field. The query remains visible and the model remains in the viewport.
- Clearing the query restores the unfiltered tree. No selection or model edit is required.

## Float and redock the drawing browser

- Context: a drawing with a sheet, border and title block; the browser is initially docked on the left.
- Trigger: drag the browser header into the drawing area, then back to the left edge.
- Feedback: the browser becomes a narrow floating panel over the sheet and keeps the drawing resources and sheet tree visible. Redocking restores the left panel and leaves the sheet in the remaining workspace. The displayed sheet scale and position adjust with the viewport.
- Limits: floating-panel resizing was not established and is not specified by this sequence.

## Close and restore the drawing browser from its tab area

- Trigger: click the close control on the docked Model browser tab, then use the remaining plus control and select Model Browser.
- Feedback: the browser tree disappears, the drawing takes the freed workspace, and a small plus control remains at the upper left. The plus opens a compact menu of available panels. Selecting Model Browser brings the left panel and drawing tree back.
- Limit: this covers the drawing workspace and one close-and-restore sequence.

## Add and switch browser panel tabs

- Context: a drawing with its Model browser docked at the left.
- Trigger: open the plus menu and select an available log panel, select Model, select the log tab, then close it.
- Feedback: the log opens in the same dock as a second tab and initially becomes active. Selecting Model reveals the drawing tree again; selecting the log tab replaces it with the log area. Closing the log tab returns the dock to the Model browser while the drawing remains visible.
- Limits: this describes panel navigation only; log contents and other panel types were not tested. Panel names are described generically.

## Original schematic

[Workspace panel states](workspace-states.svg) illustrates the observed left-docked, floating and right-docked arrangements. It is an original conceptual diagram, not a source screenshot or pixel-accurate specification.

[Panel tab states](panel-tabs.svg) illustrates the observed Model browser, active log tab and restored browser after closing the log. The artwork is original and schematic.

## Drawing browser expansion

The drawing browser menu appeared as a compact vertical list beneath the header menu control. Collapsed resource and sheet rows had expansion markers; expanded branches displayed indented child rows and connector lines. The sheet retained its appearance through both actions. The list contents are specific to the observed drawing context.

## Drawing browser search feedback

The search field appears as a narrow row beneath the browser tabs. Matching border text is accented within indented tree rows; relevant ancestor rows remain visible. An unmatched query replaces the tree with a small no-results message. The drawing sheet stays visible throughout.

## Drawing browser row selection

The selected sheet entry uses a blue accent behind its label. Selecting another entry moves the accent to that row; the light drawing sheet remains in the viewport. The observation does not establish the selected object's canvas highlight.

## Floating browser search

The floating panel retains the search control in its header. The search row and no-results message appear within the panel while the light drawing sheet remains visible behind it. Clearing restores the indented browser tree in the same floating panel.

## Drawing browser side docks

The drawing browser is a full-height vertical panel on the chosen side. The sheet is displayed in the remaining viewport, and its placement adjusts when the dock moves. Browser tree rows and the selected-row indication remained present through the observed move.

## Individual branch expansion

A small marker beside the sheet row controls the visibility of indented child rows. The resource branch remains expanded and the sheet remains visible when the sheet branch is collapsed or reopened.

## Search through a collapsed branch

The filtered result reveals and indents the matching resource below its ancestor rows, with the matching text accented. On clearing the query, the original collapsed resource row returns while the sheet remains visible.
