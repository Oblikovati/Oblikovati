# Native Ribbon Customization target rows

Fifteen current cases passed 221 steps. The assertions now inspect the
rendered target list child window. A dimension behind the dialog had
previously been counted as a command row. The cases check empty lists,
staging, reorder and selection, and changes in command display options.

The native regression also staged a command successfully after starting
from the empty 3D Model list. The Linux X11 build used `1c95d5e2a` with
the dialog row changes, working import bindings and existing local GUI
filesystem, opening and saving edits. Runs used Xvfb, Openbox and Vulkan
Lavapipe. The [input hashes](current-inputs.sha256) matched the isolated
documents and preference seed. Issue #3978 tracks strict build and runtime
provenance.
