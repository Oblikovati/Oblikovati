# Native Ribbon Customization export and restore

Two current cases passed 25 steps. The restore case applied the small
icon-only command, exported it through the native chooser, removed it
from the ribbon, and reimported it as a draft. Applying the draft
restored the icon. The missing-file case preserved the selected target
and its clean state after the picker was canceled.

The native Linux X11 build used the target list diagnostic changes
in `0d6a0c656`, the bindings in this commit, draft command-list
bindings, and existing local GUI filesystem, opening and saving edits.
Runs used Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated documents and preference seed. Issue #3978 tracks
strict build and runtime provenance.
