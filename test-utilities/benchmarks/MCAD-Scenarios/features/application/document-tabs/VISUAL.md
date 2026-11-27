# Document tabs: visual observations

## Switch documents

- Context: two saved part documents are open, one with an empty sketch and one with a rectangular sketch profile.
- Trigger: select each document tab in turn.
- Layout and feedback: a horizontal row of named document tabs appears near the bottom of the workspace. The active tab has a colored underline. Selecting a different tab changes the application title, model-browser root and viewport to that document; the other tab remains present.
- Observed color: the active underline is blue in this theme; tab text is light on dark blue-grey.
- Limits: this sequence alone does not establish tab ordering, drag-reordering, overflow behavior or persistence between sessions.

## Tab context menu

- Trigger: right-click the active document tab.
- Layout and feedback: a compact vertical menu opens above the tab with labeled commands for close variants, save and properties. Close and Save show keyboard shortcuts. Dismissing the menu leaves the active document in place.
- Original icon brief: use a fresh overlapping-sheet motif with a small title bar for document tabs; use independent shapes for close and save actions.
- Limits: the initial menu inspection exercised dismissal; subsequent observations of Close and Properties are described below.

## Tab visibility and window list

- Trigger: clear Document Tabs in the View tab's User Interface menu.
- Feedback: the bottom tab strip disappears while the current document remains visible. The menu checkmark clears. Reselecting Document Tabs restores the strip and checkmark.
- With tabs hidden, the View tab's Switch control opens a list of open windows and a More Windows entry. Choosing More Windows opens a centered Select Window dialog with a scrollable document list and OK and Cancel controls.
- Limits: the attempted selection in the extended list did not yield an unambiguous correspondence between the highlighted row and activated document, so no selection-result claim is made.

## Keyboard switching and tab preview

- With several documents open, pressing Control+Tab activated a different part and updated the title, browser and viewport. No cycling order was established.
- A document-tab hover preview appeared as a small thumbnail of the document with its name and file location. This observation does not establish preview timing or behavior for every document type.
- Original icon brief: depict a document preview using a simple outlined card and abstract model silhouette, with distinct geometry from the observed interface.

## Properties from an inactive tab

- Context: a saved part's tab is inactive while another part is displayed.
- Right-clicking the inactive tab opens its context menu without changing the active viewport or browser. Choosing Properties opens a centered, light-colored dialog identifying the right-clicked part.
- The General page presents the file name and type near the top, followed by location, size and creation, modification and access timestamps. A horizontal row of property categories sits above the fields; Close, Cancel and Apply controls sit at the bottom.
- Closing the unchanged dialog returns to the original active part. The observation does not establish the behavior of edits in the other property pages.
- Original icon brief: use a fresh outlined document with a small information marker to signify document properties.

## Close an inactive saved document

- Context: multiple parts are open, including an inactive saved part without pending changes.
- Trigger: click the small close mark at the right end of the inactive part's tab.
- Feedback: the tab disappears without a save prompt, while the active document title, browser and viewport remain displayed. The observed tab bar and status area update to reflect one fewer open document.
- Limits: this observation does not establish the close behavior for documents with unsaved edits.
- Original icon brief: use a simple independently designed crossing-stroke close mark, sized as a secondary tab action.

## Close a new empty active part

- Context: a populated part is active with other documents open; a new empty metric part is then created.
- Trigger: click the close mark on the newly created part's active tab without making edits.
- Feedback: its tab disappears without a save prompt. The part that was active before creation becomes active again, and its title, model browser and viewport return.
- Limit: this single sequence does not establish a general rule for choosing the next active tab in other tab histories.

## Closing during sketch editing

- With sketch editing active, the tab close button displayed a centered light dialog asking whether to leave the current environment and continue. It had a question symbol, message-visibility dropdown, OK and Cancel buttons. Cancel returned to the sketch environment with the document open.
- Repeating the close and accepting the environment-exit prompt returned to part modeling and displayed a separate save-changes dialog naming the part. It offered a message-visibility dropdown and Yes, No and Cancel buttons. Cancel kept the part and its sketch open.
- Limits: the visibility dropdown options and the No path were not exercised. The scenario assumes the environment-exit message is enabled.
- Original icon brief: use a newly drawn circular question indicator with a distinct stroke and proportions for confirmation dialogs.

## Cancel the first save while closing

- Trigger: close a new part containing an unsaved sketch, then choose Yes in the save-changes prompt.
- Feedback: a large file chooser opens with a suggested file name, folder navigation, a file list, preview pane and Save and Cancel buttons. Cancelling it returns to the still-open part and its sketch in the browser.
- Limits: no filename or location was accepted in this cancellation sequence.

## Save while closing

- Context: a newly created part contains an unsaved sketch, and another part was active before it was created.
- Trigger: close the new part, choose Yes in the save prompt, and accept the suggested name in Save As.
- Feedback: the file appears in the chosen folder, the new part tab disappears, and the previously active part returns to the viewport and browser.
- Reopening the saved file activates its document tab and shows the sketch in the model browser. The sketch was empty, so this does not establish persistence of sketch geometry.

## Close from an inactive tab menu

- Context: a previously saved, unmodified part is inactive while another part is displayed.
- Trigger: right-click the saved part's tab and select Close from its context menu.
- Feedback: the targeted tab disappears, while the active document's title, browser and viewport remain displayed; no save prompt appears.
- Limits: the result for an inactive document with unsaved edits was not observed in this sequence.

## Switching document types

The active tab accent moves between document tabs. The part context shows a dark sketch viewport, model tree and sketch commands; the drawing context shows a light sheet, drawing resources tree and drawing commands. The drawing browser's collapsed resource row returned on switching back.

Selecting the part tab again restored its sketch ribbon, model browser and sketch viewport.

## Tab-strip document list

A tall menu opens above the control at the end of the bottom tab strip. Window arrangement commands appear above a vertical list of document names; a small checkmark identifies the active document. Additional commands appear below the list. Selecting a listed document dismisses the menu and changes the active workspace.

[Document-list wireframe](document-list.svg) is an original schematic showing the menu grouping, active-document indicator and bottom tab strip. Labels and geometry are generic; it is not a source screenshot.

## Reorder a document tab

The active drawing tab moved among neighboring tabs when dragged along the strip. Its blue active underline and drawing workspace remained visible; dragging back restored the earlier order. No intermediate insertion marker was established.

Dragging an inactive part tab moved it within the strip and transferred the active underline to that tab; the part browser and viewport replaced the drawing workspace.

## Detached drawing window

The detached drawing appears in its own window with a title bar, drawing viewport and bottom document tab. The main workspace remains visible behind it. Window placement depends on the drag; this observation does not specify redocking.

[Detached document wireframe](detached-document.svg) is an original schematic of the observed separate drawing window over the main workspace. Its geometry and labels are generic.

## Detached tab menu

The detached document tab's context menu appears above its tab as a compact vertical list. Close, save and properties entries use small action symbols; Close and Save show shortcuts aligned to the right.

## Focus changes with a detached drawing

The main application retains the ribbon and browser while the detached window displays the drawing sheet. Focusing the sheet changes those shared controls to drawing context; reselecting a main-window part restores its tree and ribbon context.

## Reverse keyboard switching

In the observed adjacent pair, Control+Shift+Tab moved the active tab accent left and changed the browser and viewport; Control+Tab returned the accent and displayed context to the right-hand part.
