# Project workspace folder selection

The original Browse step clicked the Quick Access ellipsis. The wizard's
ellipsis can share a rendered text run with the workspace path. The revised
step finds that button in the wizard's draw list and clicks its first glyph.
The other steps target Documents in the folder tree and OK and Cancel in
the folder browser footer.

The Cancel scenario checks that Documents has a highlighted row, Drawings is
visible beneath it, and the Folder field shows Documents. After Cancel, the
wizard still shows its previous workspace and project file. The OK scenario
checks that the wizard adopts Documents and shows the project file inside it.
The original project remains active when the wizard closes.

The test build used `32df309a1` with the test changes in this commit and
existing local filesystem and sketch-marking test edits. X11 used Xvfb,
Openbox, and XTest. Wayland used isolated headless Weston with Pixman and
application frame-loop input. Both used Vulkan Lavapipe. The reports merge
individual runs of the same published feature so each scenario starts in its
own process.

All eight current Project Dialog scenarios passed on each backend, with 110
steps per backend. This includes 14 steps for Cancel and 11 steps for OK.
The scenario and fixture hashes matched before and after the runs. Strict
build and runtime provenance remains open under #3978.
[Wayland report](wayland-cucumber.json) · [X11 report](x11-cucumber.json) · [Published input hashes](current-inputs.sha256)

The published visual notes and settings icon have also changed. The feature
record includes their current hashes. The run covers the scenario behavior
and visible controls exercised by its steps.
