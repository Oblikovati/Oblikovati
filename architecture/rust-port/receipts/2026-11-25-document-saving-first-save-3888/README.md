# First save and reopening an empty part

The first two current Document Saving scenarios passed all 24 steps on Linux
X11 with Xvfb, Openbox, and Vulkan Lavapipe. Cancelling the first Save As
preserved the unsaved document and its proposed filename on the next request.
Saving and reopening a new Part restored the exact file and displayed its
empty viewport, Origin branch, and End of Part marker.

The scenario build included the committed Open dialog drawing capture changes
and existing local GUI filesystem, opening and saving edits. The published
[input checksums](current-inputs.sha256) matched the isolated copies. See the
[scenario inventory](scenario-inventory.json), Cucumber reports, and JUnit
reports. Issue #3978 tracks strict build and runtime provenance.
