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

At `c5ee038dfc27d2303c4b3c8059d588ae1427c2d7`, the two small commands
and their vertical divider passed all eight steps on both backends.
The export and restore case passed all 14 steps on both backends and
verified that importing restores the draft before Apply displays the icon.

- Divider on [Wayland](divider-wayland-report.json) and [X11](divider-x11-report.json),
  with a [Wayland capture](divider-wayland.png)
- Export and restore on [Wayland](export-wayland-report.json) and [X11](export-x11-report.json)

At `d49e62d8b8d5dbec7d93c31f0bfd9ff4b69d5670`, Shell's size checks passed
37 steps per backend. They verified that enlarging the button moves its
caption beneath the icon and reflows Chamfer, and that Large survives both
tab switches and a change to Text Off. Restoring Standard put the label
beside the small icon again.

- Size and reflow on [Wayland](shell-size-wayland-report.json) and [X11](shell-size-x11-report.json)
- Tab switches on [Wayland](shell-size-retain-wayland-report.json) and [X11](shell-size-retain-x11-report.json)
- Text Off on [Wayland](shell-size-text-off-wayland-report.json) and [X11](shell-size-text-off-x11-report.json)

The [later removal checks](../2026-11-26-ribbon-panel-retirement-3856/README.md)
cover applying an empty panel and retaining it during Shell. That fix and
the icon checks were run separately from the full suite.

The builds included the existing local GUI filesystem test-module visibility
edit. Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The [input hashes](current-inputs.sha256) matched the shared
source before and after the runs. Strict build and runtime verification
remains open under #3978.
