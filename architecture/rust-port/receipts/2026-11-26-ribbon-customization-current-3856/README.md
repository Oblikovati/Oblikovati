# Current native Ribbon Customization

The full Wayland run passed 107 of 126 executions, with 1,516 steps in
passing executions. It used `a98ccab593687cbe0c7deba16467d1408783dd59`.
The [full report](full-wayland-report.json) retains the 19 failures and the
[resolved arguments](full-wayland-arguments.json) preserve source lines and
outline examples.

The centered User Commands caption and the ribbon tab bound the icon check
at `1c523e8223c0d3c8be6e3eeb875afb6e24b29bbf`. The caption case passed 11
steps per backend and the large icon without text passed eight. The checks
read the submitted icon rectangles and rendered command labels.

- Small and large caption placement on [Wayland](caption-wayland-report.json)
  and [X11](caption-x11-report.json)
- Large icon without text on [Wayland](icon-only-wayland-report.json)
  and [X11](icon-only-x11-report.json)

The [later removal checks](../2026-11-26-ribbon-panel-retirement-3856/README.md)
cover applying an empty panel and retaining it during Shell. That fix and
the icon checks were run separately from the full suite.

Both builds included the existing local GUI filesystem test-module visibility
edit. Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The [input hashes](current-inputs.sha256) matched the shared
source before and after the runs. Strict build and runtime verification
remains open under #3978.
