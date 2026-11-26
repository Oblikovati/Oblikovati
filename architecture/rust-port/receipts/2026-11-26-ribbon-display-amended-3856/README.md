# Amended Ribbon Display scenarios on Wayland

The published feature ran in 52 separate native Wayland windows.
Twenty-seven scenarios and their 312 steps passed. The new ribbon
height and compaction scenario also passed on X11. See the
[focused sizing receipt](../2026-11-26-ribbon-height-compaction-3856/README.md).

Nine cases called X11-only window helpers from a Wayland window.
Three cases stopped at an undefined step, including the new control
alignment scenario and both Shell workflows. The other failures
concern tab activation, temporary panels, or Home and document context
switching. The report retains the first failing assertion for each case.

The amended feature supersedes the source used for the
[earlier X11 acceptance](../2026-11-25-ribbon-display-current-3856/README.md).
The current runs used application revision `9a55e493c`, the new sizing
bindings, and existing local GUI filesystem edits. They ran in nested
Weston 13 with Xvfb and Vulkan Lavapipe. Strict build and runtime
provenance remains open under #3978.

- [Wayland report](wayland.json) and [arguments](wayland-arguments.json)
- [Published input hashes](inputs.sha256)
