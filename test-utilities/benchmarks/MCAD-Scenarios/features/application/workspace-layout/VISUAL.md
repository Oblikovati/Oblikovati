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

