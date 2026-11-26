# Current File Menu scenarios on Wayland

All eight published scenarios and 67 steps passed in separate native
Wayland windows with disposable profiles. The cases verify the menu over
Parts, a Drawing, and Home. Escape and the outside click preserved the
active document. The recent list displayed its document type, name, and pin.
Save As and New showed their expected submenu entries.
[Cucumber report](cucumber.json) · [Published input hashes](inputs.sha256)

The feature progress checker matched the reports and captured arguments
against the published declarations. The test binary used application commit
`95515bee5` and existing local GUI filesystem edits. The runs used headless
Weston with scripted input, Pixman, and Vulkan Lavapipe. Issue #3978 tracks
strict build and runtime provenance.
