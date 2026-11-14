# Open document tabs

The MCAD Software presents open documents as tabs along the workspace edge. Selecting a tab activates its document and updates the viewport and model browser. With multiple documents open, Control+Tab was observed to activate another document; the traversal order was not established.

The close button on a saved inactive tab closes that document while the active document remains displayed. Closing a newly created, unedited active part returned to the part that had been active before its creation. A tab context menu provides close, save and properties commands. Choosing Close from a saved inactive tab's menu closes that tab while the current document remains active. The Properties command can inspect the document associated with an inactive tab while the current document remains active. Document tabs can be hidden and restored from the View interface menu. When hidden, open windows remain accessible through the Switch menu and its extended list.

The executable-format behavior specifications are in [document-tabs.feature](document-tabs.feature). The visual layout and original icon briefs are in [VISUAL.md](VISUAL.md). Scenarios tagged `@needs_review` or `@needs_fixture` identify further specification or fixture work; no step definitions are supplied.

Observed coverage does not establish keyboard traversal order, tab order persistence across sessions, or the result of choosing a document in the extended window list.

When the environment-exit message is enabled, attempting to close a part while editing a sketch asks whether to exit that environment. Cancel keeps the sketch active. Continuing with a newly created unsaved sketch leads to a separate save-changes prompt; cancelling that prompt keeps the part open in the modeling environment.

Choosing to save a never-saved part from the close confirmation opens a Save As dialog. Cancelling that dialog leaves the part open.

In the observed new-part sequence, accepting the suggested Save As name created a part file, closed the tab, and returned to the previously active part. Reopening that file showed its sketch in the model browser.

## Switch between drawing and part contexts

With a drawing active and its resource branch collapsed, selecting an already-open part tab displayed that part's sketch editing viewport, model tree and sketch ribbon. Selecting the drawing tab restored the drawing sheet, drawing tree and drawing ribbon; the resource branch remained collapsed. Both documents remained open. This observation covers one in-session switch and does not establish layout persistence after reopening the application.

Returning to the part tab after visiting the drawing restored the sketch viewport and sketch ribbon, with the same sketch editing environment still active.

## Select a document from the tab-strip list

With many documents open and a part active, the menu at the end of the tab strip listed the open documents and marked the active part. It also presented window arrangement and close commands. Selecting the drawing from the list activated its sheet and browser; the part stayed open. Arrangement and close commands were not invoked in this observation.

## Reorder an active document tab

Dragging an active drawing tab left across several neighboring tabs moved it within the tab strip. The drawing sheet and browser remained active and visible. Dragging it back restored its earlier position. This observation does not establish tab-order persistence across sessions.

## Reorder an inactive document tab

With a drawing active, dragging an inactive part tab left moved the tab among its neighbors and activated the part, displaying its model viewport and browser. The drawing remained open. The part tab was moved back to its former position, and the drawing was reselected. This does not establish behavior for all document types or session persistence.

## Detach a drawing tab

Dragging the active drawing tab upward from the main tab strip into the workspace displayed the drawing in a separate top-level window with its own tab. The part remained open in the main workspace. The detached window initially appeared partially off screen and was moved into view using its window controls. Returning it to the main tab strip was not established.

## Detached tab context menu

The drawing tab in the detached window exposes a context menu with close variants, save and document properties, including displayed Close and Save shortcuts. Dismissing the menu left the detached drawing open. No menu command was invoked in this observation.

After detachment, selecting a different open part tab activated that part in the main workspace while the drawing window remained open separately.

## Shared controls and detached document focus

With a part active in the main workspace and a drawing in a separate window, clicking the drawing sheet changed the main application's browser to the drawing tree and its ribbon to drawing-context tabs. Selecting the part tab in the main workspace restored the part browser and part-context ribbon. The detached drawing remained open. This describes the observed focus sequence, not session persistence.

## Reverse and forward keyboard switching

With two neighboring part tabs visible and the right-hand part active, Control+Shift+Tab activated the left-hand part. Control+Tab then returned to the right-hand part, restoring its browser and viewport. This single adjacent pair does not establish wraparound or behavior across every document arrangement.
