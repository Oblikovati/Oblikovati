# First save: visual observations

- Context: a newly created part without a saved path.
- Trigger: press Control+S.
- Layout: a modal save dialog presents navigation and file listing on the left and center, a preview and related options on the right, and file name, file type, Save and Cancel controls along the bottom.
- Feedback: the proposed file name is selected for replacement. In this observation the preview-picture option is checked and an isometric preview option is selected. Cancel closes the dialog and leaves the part active.
- Original icon brief: use a fresh storage-tray shape with a descending document corner, legible at small sizes.
- Limits: the initial location, proposed name and option defaults are context dependent. Overwriting and saving a modified existing document were not exercised in that sequence.

## Save, close and reopen an empty part

- Trigger: accept a non-existing proposed file name, close the saved document tab, then open that file from the same folder.
- Feedback: the save dialog closes and the document tab and browser root display the saved file name. Closing the tab returns to another open document. The Open dialog offers location and file selection with a preview area; opening the file restores the empty part and its browser.
- Limits: this establishes a same-session round trip for an empty part. It does not verify persistence of modeled geometry, modified documents, or a restart of the software.


## Save an edited part and reopen it

- Context: an already saved part was modified by adding a planar rectangular sketch and finishing the sketch.
- Trigger: save the modified document, close its tab, and reopen the same file in the session.
- Feedback: the save returns to the workspace without showing Save As. The Open dialog displays a thumbnail of the sketch. After reopening, the browser contains the sketch and its rectangular outline is visible in the viewport at a different view scale.
- Limits: the sequence establishes visible sketch persistence for this fixture; exact dimensions, constraint preservation, solid geometry and restart persistence were not measured.

## Preview options in Save As

- Trigger: clear and reselect Save Preview Picture, then select another preview source.
- Feedback: clearing the checkbox replaces the preview with a neutral image placeholder and hides the source radio buttons. Reselecting the checkbox restores the preview and source choices. Selecting Active Window on Save moves the radio selection away from Active Component Iso View on Save. The initial choice was restored before cancellation.
- Limits: the dialog was cancelled; this sequence does not establish the saved image contents or persistence of the option.

## Import a preview image: browse and cancel

- Trigger: select Import From File while preview saving is enabled, open its browse control, and cancel the image chooser.
- Layout and feedback: a browse button appears next to the selected radio option, while the preview region shows a placeholder. The browse button opens a nested image-file chooser above Save As. Cancelling the chooser returns to Save As with Import From File still selected and the placeholder still shown. The prior isometric preview choice was restored before cancelling Save As.
- Limits: no image was imported and no file was saved in this sequence; image formats and saved-preview contents were not verified.

## File type list

- Trigger: open the file type list at the bottom of Save As for a new part.
- Feedback: a short list offers the native part-file type and an All Files filter. The dialog remains open while the list is displayed.
- Limits: changing the filter or saving under an alternative extension was not established.

## Close with unsaved changes

- Context: a new unsaved part contains a finished empty sketch and the prompt setting is Always show this message.
- Trigger: close the active document tab.
- Layout and feedback: a centered modal asks whether to save changes and offers Yes, No and Cancel. A drop-down presents prompt-frequency choices. Dismissing that list without selecting a different option and choosing Cancel leaves the document and sketch visible.
- A second close attempt followed by Yes opens Save As; cancelling Save As also leaves the modified document open.
- Limits: the No branch, altered prompt-frequency choices and their persistence were not exercised.

## Save and close a newly modified part

- Trigger: close an unsaved part containing a finished empty sketch, choose Yes, and accept a proposed non-existing filename in Save As.
- Feedback: Save As closes, the document tab closes, and another open document becomes active. Reopening the newly saved file restores its sketch entry in the browser.
- Limits: this sequence establishes the close-initiated save for an empty sketch in the same session; modeled geometry and later application restarts were not checked in this sequence.

## Original visual assets

- [Conceptual save dialog wireframe](save-dialog-wireframe.svg) records the major regions and controls. It is a schematic, not a screenshot or a pixel-exact specification; preview choices depend on the current selection.
- [Save icon](save-icon.svg) is a newly drawn, colored document-and-tray symbol based on the icon brief above.

## Capture the active window preview

- Trigger: in Save As for a new empty part, select Active Window and activate its adjacent capture control.
- Feedback: selecting Active Window displays a neutral image placeholder and a small action icon to its right. Activating the icon replaces the placeholder with a dark preview matching the empty viewport while the dialog remains open.
- Original icon brief: use a small outlined view frame with a distinct capture dot, designed independently.
- Limits: the dialog was cancelled; persistence and capture contents for non-empty models were not tested.

## Select an existing file in Save As

- Trigger: select a listed existing part while saving a new empty part.
- Feedback: the selected row gains a blue highlight, the bottom file-name field changes from the proposed new name to the selected file's displayed name, and the status area reports one selected item. Cancel returns to the unsaved part.
- Limits: Save was not activated after selecting the existing file; replacement behavior was not observed.

## Navigate and cancel a save location

- Trigger: open a visible subfolder in the first-save dialog, then use Back or cancel and reopen Save As.
- Feedback: the breadcrumb and file listing update for the subfolder while the proposed filename remains in the field. Back restores the previous folder listing. After cancellation in the subfolder, a fresh save attempt for the same unsaved part reopened at the initial folder with the proposed name selected.
- Limits: one folder and part were observed in a single session; behavior after saving or restarting was not checked.

## Cancel changes to preview settings

- Starting with preview saving enabled and the isometric source selected, clear the checkbox, cancel, and reopen Save As for the same new part: the checkbox is selected and the preview source choices return.
- Starting from the same initial choice, select Active Window on Save, cancel, and reopen: the isometric source is selected again.
- Limits: both observations concern the same unsaved part and session; they do not determine behavior after a successful save or restart.
