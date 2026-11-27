# Native Model Browser feature and docking checks

Sixteen current cases passed 184 steps. They verified Find and saved-search
cancellation, sketch inputs and dimensions, origin and Favorites controls,
feature visibility, the Model Browser's dock positions, and representation
branches in the empty Part.

The Linux X11 build included the scenario bindings in `6d0f4212f` and
existing local GUI filesystem, opening and saving edits. Runs used Xvfb,
Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict build
and runtime provenance.
