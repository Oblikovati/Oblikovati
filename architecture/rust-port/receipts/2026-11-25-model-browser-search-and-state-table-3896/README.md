# Native Model Browser search and model-state table

Fourteen current cases passed 211 steps. They checked search grouping,
conditional suppression and adaptive option cancellation, creating and
switching model states, and applying or canceling model-state table
property edits.

The Linux X11 build included the scenario bindings in `6d0f4212f` and
existing local GUI filesystem, opening and saving edits. Runs used Xvfb,
Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict build
and runtime provenance.
