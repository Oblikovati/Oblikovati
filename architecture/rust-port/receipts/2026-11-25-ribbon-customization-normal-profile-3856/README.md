# Native Ribbon Customization configuration seed

Fourteen executions across the first thirteen published declarations passed
136 steps. Both file-dialog outline examples passed. These are the first
native runs with the published Normal ribbon configuration loaded at startup.

Each case copies the seed into a new writable temporary directory and
restarts the native test window with that directory. Document fixtures
open after the restart. The published and isolated seed hashes matched
after testing. The results cover opening Customize, pending and applied
changes, row order, command appearance, and file dialogs.

The Linux X11 build used `154ab2fbe` with the test startup changes and
existing local GUI filesystem, opening and saving edits. Runs used Xvfb,
Openbox and Vulkan Lavapipe. See the [input hashes](current-inputs.sha256)
and [scenario inventory](scenario-inventory.json). Issue #3978 tracks
strict build and runtime provenance.
