# Native Ribbon Customization drafts and import

Thirteen current cases passed 191 steps. They covered staged command edits,
applying and canceling drafts, importing settings, validating malformed
input, and retaining ribbon settings while using the native dialog.

The Linux X11 build used the disposable profile setup in `b1373d818`
and existing local GUI filesystem, opening and saving edits. Runs used
Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated documents and preference seed. Issue #3978 tracks
strict build and runtime provenance.
