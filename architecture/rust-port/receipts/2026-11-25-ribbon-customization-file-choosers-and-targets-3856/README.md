# Native Ribbon Customization file choosers and targets

Nineteen executions across eighteen current declarations passed 289 steps.
They covered opening and canceling file choosers, preserving pending
customization across native dialogs, and command target or ribbon changes.
Both Import and Export examples in the file-chooser outline passed.

The Linux X11 build used the disposable profile setup in `b1373d818`
and existing local GUI filesystem, opening and saving edits. Runs used
Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated documents and preference seed. Issue #3978 tracks
strict build and runtime provenance.
