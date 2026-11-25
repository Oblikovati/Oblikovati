# Home Recent search, history, views, and assembly filter

Four current scenarios passed. Search showed only Part26 and restored
all five published rows when applied again with a cleared field.
Search History showed the Part26 entry with its history mark. Switching
to Grid checked the actual names and modification dates for the five
cards, and returning to List restored the metadata columns. Filtering
to Assembly left one row and clearing the filter restored the Parts.

The test recognizes clipped assembly and drawing labels by their
visible prefix and the document-kind mark on the same row.

The native runs used base `6eb94e2740e739a595f1782a2a5c2173738c5b7e`, the current bindings, and
existing local GUI filesystem, opening and saving edits. The desktop
used Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. The published
[input hashes](current-inputs.sha256) matched the isolated copies.
Strict build and runtime provenance remains open under #3978.
