# Model browser root context

On 2026-11-19, the focused scenario passed all 11 steps under Xvfb and Vulkan
lavapipe. The feature SHA-256 was
`603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
The test selected the actual root caption within Model, verified its row and
solid-edge highlights, inspected the menu and cleared selection.

A separate native X11 pointer test produced the selected and cleared captures.
It checked that Copy File Path is available for the saved fixture. Open File
Location, Create Drawing View and Open Drawing remain disabled; this report does
not claim those actions work. The source directory is unversioned, and this is
not a full-feature run or strict verified acceptance. See issue #4116 and
issue #3978.
