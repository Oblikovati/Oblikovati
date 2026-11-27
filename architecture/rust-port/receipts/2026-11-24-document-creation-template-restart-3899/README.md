# Native template selection across restart

The scenario passed all 22 steps. After reopening the same desktop profile
with a fresh native GUI and owner, the chooser selected the Metric
`Standard (mm)` file and showed millimeter units and its stored Generic
material. Create and Cancel were available. Escape closed the chooser,
leaving the saved Part and its visible Sketch1 active.

The [Cucumber report](visible/cucumber.json) and
[final frame](visible/final.png) came from the code in
`ff684bda6b276bb7a45c4421e809424186baa2ae` with existing local GUI filesystem,
opening, and saving edits. The run used Linux x86-64, Xvfb, Openbox, and Vulkan
Lavapipe. The [input hashes](current-inputs.sha256) matched the shared source
and its isolated copy after the run. Strict build and runtime provenance
remains open under #3978.

A separate check launched the desktop executable twice with the same
disposable profile. The saved path matched the supplied template file.
The [selection and details](desktop-processes/before.png) matched the
[restored chooser](desktop-processes/after.png), including the selection
highlight. Both fixture hashes matched after the run.
[Process evidence](desktop-processes/process-restart.json)

The [process check source](desktop-processes/verify_process_restart.py)
takes the repository, desktop executable, project file, Part file, and an
unused output directory. It runs under a 1440 × 900 X11 display.

The 38 focused GUI tests passed, including rejecting a matching title from
another catalog. Clippy and formatting checks passed.
