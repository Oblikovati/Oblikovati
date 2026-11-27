# Current Document Saving scenarios on Wayland

All 23 dependency-ready scenarios and 260 steps passed. The first-save
preview checks covered switching sources, importing an image, capturing the
active window, and the shaded preview of the 40 × 25 × 20 mm block.
The folder cases ran from fresh writable copies of the published
`SaveAsNativeFolder`. They covered folder history, navigation modes,
file-list presentation, filename retention, and overwrite refusal.
[Cucumber report](cucumber.json) · [Published input hashes](inputs.sha256)

The first run showed a selected Active Window radio and an empty ISO radio
in the captured frame. The old probe sampled the unchecked circle and the
surrounding background and reported both as selected. The new probe reads
the rendered checkmark-colored dot. The Browse test also saw both labels in
one text run, so it now checks their order and clicks inside the action
caption. The exact-solid case uses the existing Distance A entry helper
before checking the solid dimensions and preview.
[Radio baseline](radio-baseline.json) · [Browse baseline](browse-baseline.json)

The radio probe passed a focused test at 100% and 150% scale. On X11, the
standalone preview cancellation regression and the published Find criteria
grouping scenario also passed.

The generated-part case requires Assembly Simplify to create an unsaved Part
with an assigned output path and a source Assembly reference (#3963).
Its setup step remains unbound in the current run.

The native build used `6d0932a49` with the verifier changes and existing local
GUI filesystem edits. Each run used a separate headless Weston session with
scripted input, Pixman, and Vulkan Lavapipe. Issue #3978 tracks strict build
and runtime provenance.
