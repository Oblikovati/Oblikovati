# Home Recent search recovery and selection filters

Five current scenarios passed, and the search and history recall
regressions passed again. Empty search results recovered to the full
list. Clearing search and filters restored all entries, and clearing
only the type filter preserved the Part26 query. The tests checked
selection across List and Grid and verified that filtering a checked
Part out of view clears its selection.

The native runs used base `0781f3cb1036ec3f0ef4fc272eb36ef337434e93`, updated bindings, and
existing local GUI filesystem, opening and saving edits. The desktop
used Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. The published
[input hashes](current-inputs.sha256) matched the isolated copies.
Strict build and runtime provenance remains open under #3978.
