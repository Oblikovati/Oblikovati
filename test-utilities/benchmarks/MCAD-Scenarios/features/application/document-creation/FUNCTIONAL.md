# Create a document from a template

With a document open, Control+N opens the MCAD Software's new-document chooser. The chooser organizes templates by folder and document type. Selecting a part template displays its details, including its units: the observed standard metric and inch templates reported millimeters and inches respectively. Available templates depend on configuration.

Choosing Create for the standard metric part opens a new active part tab. Its viewport initially contains no visible model geometry, and its browser includes an origin and an end-of-part marker. Cancelling the chooser leaves the previously active document displayed.

The observed behavior is expressed in [document-creation.feature](document-creation.feature); [VISUAL.md](VISUAL.md) describes the chooser's layout and an original icon brief. The scenarios remain tagged for review.

Selecting the standard metric assembly template and choosing Create opens a new empty assembly tab with assembly-specific ribbon commands and browser entries for representations, relationships and origin.

A drawing template can create a drawing document with a sheet, border and title block, depending on template content. The drawing workspace provides view commands and lists the sheet and its border in the browser.

Selecting the metric sheet-metal template creates an empty part in the sheet-metal workspace, with a folded-model branch and origin in the browser.

Template gallery groups can be collapsed and expanded. In the observed part group, collapsing hid its tiles but retained the selected template details; reopening showed the same selected tile.

Switching template folders updates the gallery. In the observed metric-to-English sequence, the details pane retained the previous template until a tile in the new folder was selected.

Creating a presentation from the observed metric template immediately opens an assembly insertion dialog. Cancelling leaves a presentation with an empty scene, a storyboard timeline and a snapshot views panel.

The template display menu provides several icon sizes, List and Details, and a Grouped by Type toggle. Choosing Details changes the template gallery to rows with Name, Size and Type columns; the separate details pane remains visible.

## Toggle template grouping

In the metric template folder, clearing Grouped by Type changed the gallery from type sections to a flat collection of template icons. The details pane continued to show the previously selected metric part's information. Selecting the option again restored the type headings and retained the displayed details. This does not establish selection persistence after the chooser closes.

## Select a template in list view

With type grouping enabled in the metric folder, selecting List displayed compact template entries under type headings while retaining the details pane. Selecting the standard metric assembly entry updated the details to identify an assembly with millimeter units. Returning to Medium icons changed the gallery presentation while the assembly details remained displayed. The chooser was canceled afterward; no document was created.

## Template folder history and reopening

After selecting the English folder and then the metric folder, Back returned to the English folder and its gallery; Forward returned to the metric folder and its gallery. In a separate immediate sequence, canceling the chooser with a standard metric assembly selected and reopening it in the same session restored the metric folder, selected assembly tile and assembly details. Behavior after application restart was not observed.

## Template chooser Home

From the metric folder, the navigation bar's Home control selected the template root. In the observed setup the root contained no direct template entries, so the type sections were empty, while the previously selected metric assembly details remained visible. Back returned to the metric folder and its gallery. Root contents depend on the template configuration.

## Template icon size

In the grouped metric gallery, switching from medium to extra-large, then small and back to medium changed the size and arrangement of template entries. The previously selected metric assembly information remained in the details pane through these changes. The chooser was canceled after restoring medium icons. Persistence across sessions and individual thumbnail contents were not established.

## Create by double-clicking a template

Double-clicking the standard metric part template closed the chooser and directly created an active empty part with its own tab, origin and end-of-part browser marker. The new unedited part was closed afterward, returning to the prior document. This observation covers the part template only.

## Create a selected template with Enter

After clicking the standard metric part template, pressing Enter dismissed the chooser and created a new active empty part. The unedited part was closed afterward. This sequence does not establish keyboard activation for other templates or focus states.

## Dismiss the chooser with Escape

Pressing Escape while the template chooser was open closed it and returned to the previously active part, with its browser, viewport and existing tabs displayed. No new document appeared.
