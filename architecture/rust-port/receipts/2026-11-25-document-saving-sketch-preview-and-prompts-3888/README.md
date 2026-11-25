# Saving a sketch, preview controls, and close prompts

Nine current Document Saving scenarios passed all 103 steps on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. A writable copy of the published empty
Part retained the rectangular sketch after Control+S, close, and reopen.
The preview source controls, image chooser cancellation, native file-type
menu, and Active Window preview all behaved as specified.

The close scenarios verified that cancelling the prompt or Save As kept the
modified Part open with its sketch. Saving from the close prompt activated the
published `PartOne.opd` tab, and reopening the newly saved Part restored its
finished sketch.

The runs used the corresponding bindings and existing local GUI filesystem,
opening and saving edits. The published [input checksums](current-inputs.sha256)
matched the isolated copies. See the [scenario inventory](scenario-inventory.json)
and individual Cucumber and JUnit reports. Issue #3978 tracks strict build and
runtime provenance.
