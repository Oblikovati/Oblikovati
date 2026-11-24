# Current native application closing

The complete current feature passed seven scenarios and 51 steps.
The existing-path Save case changed the staged Part and showed its new
XY sketch on reopening. Discard preserved the staged file byte for byte
and the reopened Part had no new sketch. Both cases waited for the native
window loop to exit and reopened the file in a fresh window and owner
within the same test process. Cancelling the first Save As preserved the
same unsaved metric Part and its sketch.

The run also covered cancelling the dirty Part and Assembly prompts,
closing an unchanged Assembly and closing from Home. Successful exits
without a remaining document have a frame from immediately before Close.

[Results](cucumber.json), [JUnit](junit.xml), [scenario arguments](scenario-arguments.json),
and [input hashes](current-inputs.sha256) record the run. The input hashes
matched the shared source and isolated fixture copy before and after the run.

The build used base revision `6429cc234683bc3454dd11de4c605438058321dc` with the
bindings in this change and existing local GUI filesystem, opening, and
saving edits. The environment was Linux x86-64, Xvfb, Openbox, and Vulkan
Lavapipe. Strict build and runtime provenance remains open under #3978.
