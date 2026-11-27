# Current Application Closing feature

All seven published scenarios and 51 steps passed individually on both
Wayland and X11. The checks covered cancel with a modified Part or
Assembly, save or discard and reopen an existing Part, cancel the first
Save As, and close from Home or an unchanged document without a prompt.
[Wayland report](wayland.json) · [X11 report](x11.json)

The Assembly setup now clicks the Browse text at the end of the rendered
label when the draw packet groups it with Component. Both original runs
stopped before the file chooser. The corrected step opens the published
Part through the chooser and places it in the Assembly. Escape dismisses
the close prompt and leaves the edited Assembly open.
[Initial Wayland run](assembly-before-wayland.json) · [Initial X11 run](assembly-before-x11.json)

The cases ran with `OBK_BDD_FEATURE=application-closing` and the exact
published name in `OBK_BDD_SCENARIO`, using
`application_scenarios_use_the_native_window --test-threads=1`.
Wayland used headless Weston with the Pixman renderer and delivered pointer,
keyboard, and compositor close events through the frame loop. X11 used Xvfb
and Openbox, with XTest for the desktop close shortcut and prompt input.
Both used Vulkan Lavapipe. The private Weston process exited with SIGSEGV
during multiple consecutive-window runs, including one with Pixman.
The first five passing results came from a shared Pixman process. Each
of the two clean-close scenarios passed on a fresh Pixman process.
[Interrupted Save run](weston-surface-loss.json) · [Interrupted clean-close run](weston-pixman-surface-loss.json)

The existing [file placement test](assembly-placement-regression.txt)
also passed on X11 with the corrected Browse click.

The binary used `afced5c5f` with the test changes in this commit and existing
local GUI filesystem and Ribbon Customization edits. The published
[input hashes](current-inputs.sha256) matched before and after. The shared
scenario mount is unversioned. Strict provenance remains open under #3978.
Earlier focused receipts cover [dirty Part Cancel](../2026-11-25-application-closing-wayland-4030/README.md),
[save and discard](../2026-11-25-application-closing-existing-part-4030/README.md),
and [Escape in the first Save As](../2026-11-25-application-closing-first-save-4030/README.md).
