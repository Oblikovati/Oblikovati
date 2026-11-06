# Save and reopen a part document

For a new unsaved part, Control+S opens Save As with a proposed file name selected, a location and file-type selector, and preview options. Cancel returns to the active unsaved document. Accepting a non-existing name saves the part and updates the document tab and browser root. An empty part saved, closed, and reopened in the same session retained its empty workspace and origin.

Saving an already saved part after adding and finishing a rectangular sketch returned directly to the workspace without opening Save As. Closing and reopening it in the same session restored the sketch entry and visible profile.

Save Preview Picture controls whether preview source choices are displayed. The observed choices include an isometric component view, the active window, and an imported file. Choosing import exposes a browse control; cancelling the nested image chooser retains the import selection. The save dialog can be cancelled without saving. The part file-type list offered the native part type and an All Files filter.

With save-change prompts configured to always show, closing an unsaved modified part opens a Yes/No/Cancel prompt. Cancel keeps the part open. Yes opens Save As; cancelling that dialog also keeps the part open. Accepting a non-existing filename saves and closes the document, and reopening it in the same session restored its sketch entry. The prompt also exposes frequency choices; their effects were not tested.

These observations are limited to the described part fixtures and same-session reopen sequences. They do not establish overwrite behavior, imported preview contents, prompt preference persistence, or restart persistence. See [document-saving.feature](document-saving.feature) for review-tagged scenarios and [VISUAL.md](VISUAL.md) for the observed layouts and original icon brief.

For an empty unsaved part, selecting Active Window in the preview options displayed a placeholder and a capture control. Activating that control replaced the placeholder with a dark view matching the empty workspace. The separate Active Window on Save option was not used in this sequence. The dialog was cancelled, so saved preview contents were not checked.

In the first-save dialog, selecting an existing part in the file list replaces the proposed file name with the selected file's displayed name. Cancelling leaves the new part active; replacement of an existing file was not attempted.

In the observed first-save dialog, opening a subfolder changes the location breadcrumb and file list while retaining the proposed filename. Back returns to the previous folder. After navigating into the subfolder and cancelling, reopening Save As for the same unsaved part returned to the initial folder with the original proposed name selected. This is a same-session observation, not a general persistence guarantee.

For the observed new part, cancelling Save As discarded changes to the preview controls: on reopening, Save Preview Picture was selected again after it had been cleared, and the original isometric source was selected again after choosing Active Window on Save. Both checks were performed in the same session without saving.
