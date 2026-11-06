# Document creation: visual observations

## Template chooser

- Trigger: press Control+N with a part document open.
- Layout: a modal window overlays the workspace. A folder tree is on the left, a scrollable gallery of templates grouped by document type occupies the center, and a details and preview area is on the right. Project controls and Create and Cancel buttons are at the bottom.
- Feedback: the selected gallery item has a blue selection highlight. Its details include the template name, document type, units and material. Choosing another folder replaces the center gallery; the previously selected template details remain visible until a new template is selected in the observed sequence.
- Observed color: light dialog surfaces, dark text, blue selection and blue part-template icons. These are local observations.
- Original icon brief: use a fresh sheet outline with a small geometric solid, varying the overlay shape by document category rather than copying the template icons.
- Limits: template lists and default selection depend on available configuration; no persistence beyond the observed reopening is asserted.

## New metric part

- Trigger: select the standard metric part template and choose Create.
- Layout and feedback: the dialog closes, a new tab becomes active, and the empty viewport and model browser appear. The browser includes an origin and an end-of-part marker.

## New metric assembly

- Trigger: select the standard metric assembly template and choose Create. The template details identify an assembly and millimeter units.
- Feedback: the chooser closes, an assembly tab becomes active, and the empty viewport appears. The left browser presents assembly and modeling views, with representation, relationship and origin entries. The ribbon changes to assembly-oriented controls including component placement and creation.
- Observed color: the active tab uses a blue underline; unavailable commands are muted in the dark ribbon.
- Original icon brief: use two independently drawn interlocking geometric pieces to represent an assembly template, with a different arrangement from the observed source.
- Limits: no components were placed and no save behavior was exercised.

## New drawing from a sheet template

- Context: a drawing template with a blank sheet, border and title block is selected in the template chooser. Its preview area provides a Use Template choice and additional example previews.
- Trigger: choose Use Template and Create.
- Feedback: a new drawing tab is active, a light sheet fills most of the dark viewport, and the sheet displays a perimeter border and a title block in the lower-right corner. The left browser lists drawing resources and the sheet, with a border entry beneath it. The ribbon changes to drawing view commands.
- Original icon brief: use a newly drawn sheet outline with a small abstract view and title area, arranged differently from the observed template icons.
- Limits: sheet format and title block depend on the selected template; no model view was placed.

## New metric sheet-metal part

- Trigger: select the metric sheet-metal template in the part group, then choose Create. Its details identify a part and millimeter units.
- Feedback: a new part tab becomes active with an empty viewport. The browser has a folded-model branch containing an origin and an end marker. The active ribbon provides sheet-metal controls, with unavailable commands muted.
- Original icon brief: represent sheet-metal creation with a distinct folded-strip silhouette rather than reproducing the observed source symbol.
- Limits: no sheet-metal geometry was created and no template settings beyond the displayed units were verified.

## Collapse and expand a template group

- Trigger: use the disclosure control on the part group in the template gallery.
- Feedback: collapsing hides that group's tiles and shifts subsequent groups upward. The selected sheet-metal template's details remain in the right pane. Expanding the group restores its tiles and selected highlight.
- Limits: this sequence covers the part group within one chooser session; persistence and other groups were not checked.

## Switch template folders

- With a metric sheet-metal template selected, selecting the English folder replaces the gallery with that folder's available tiles. The details pane continues to show the previous metric template and millimeter units until an English template is selected.
- Selecting the inch sheet-metal tile updates the details pane to that template and inch units.
- Limit: this records the displayed selection and details only; no document was created after switching folders.

## New presentation

- Trigger: select the metric presentation template and choose Create. The selected template is described as a presentation for assembly animation and exploded views.
- Feedback: a new presentation opens and immediately presents an Insert file chooser filtered to assembly files. Cancelling leaves an empty scene, with a model browser on the left, a snapshot views panel on the right, and a storyboard timeline beneath the viewport. The presentation ribbon offers Insert Model.
- Original icon brief: use a distinct set of separated simple solids and a motion cue to indicate an assembly presentation.
- Limits: no assembly was inserted and no animation or saved presentation was observed.

## Template display menu and details view

- Trigger: open the display control at the right end of the chooser navigation bar and select Details.
- Feedback: a compact vertical menu offers icon sizes, List, Details and a checked Grouped by Type option. Details replaces the tile grid with compact rows under Name, Size and Type headings; the selection information panel remains to the right.
- Original icon brief: a short stack of lines alongside a small downward chevron can communicate display options.
- Limits: other view modes and grouping changes were not exercised.

## Template grouping toggle

The display menu uses a checkmark for Grouped by Type. When cleared, the central gallery shows template icons without group headings; the right details pane remains visible. Restoring grouping brings back labeled sections with disclosure controls.

## List view template selection

List view uses small document-type icons beside compact names within the expanded group sections. Selecting an entry highlights it and updates the right-side details and illustration. Switching back to medium icons restores larger gallery tiles while retaining the displayed details.

## Template folder history

Back and Forward controls appear in the chooser's top navigation bar. Navigating history changes the highlighted folder in the left tree and the central gallery. Reopening after the observed cancellation restored the highlighted template and its details pane.

## Template root navigation

Home changes the highlighted tree row to the template root. With no direct templates in that root, the grouped gallery shows headings above empty areas and retains the previous details pane. Back restores the earlier folder gallery.

## Icon-size changes

Extra-large mode showed large preview tiles with fewer entries in the visible gallery; small mode showed compact icons and more entries. Type headings and the right details pane remained. Medium mode restored the earlier gallery scale.

## Double-click a part template

Double-clicking the metric part tile dismissed the chooser and revealed a new dark, empty part viewport with its model browser and active document tab.

## Keyboard template creation

After selecting the metric part tile, Enter replaced the chooser with an empty part viewport, model browser and active document tab.

## Escape from the chooser

Escape dismissed the centered chooser, revealing the previously active part workspace and its existing document tabs.
