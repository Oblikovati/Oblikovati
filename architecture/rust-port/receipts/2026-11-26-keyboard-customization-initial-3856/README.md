# Keyboard customization initial state

Customize starts each opening with an empty keyboard search and both
catalog filters set to All. Reopening preserves the selected tab.

The published scenario opens the Part from the named preference profile
and checks the keyboard tab, both filters, the empty search, and the
unselected command and shortcut fields. The fixture has no sketch in
edit. The close step verifies that the original Part stays active and its
sketch edit state stays unchanged.

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Inspect the keyboard customization initial state | [16 steps passed](wayland/cucumber.json) | [16 steps passed](x11/cucumber.json) |

The baseline run stopped at the Categories step on both backends because
the diagnostics combined adjacent labels into a single text run. The
updated check verifies both filter values in the rendered row. A separate
native test exercises Search, Clear, staged Reset All Keys, Cancel, and
Apply, including the filter state on reopening. [Test results](test-results.txt)

The build used `83ee4fb90` with these changes and existing local filesystem
and marking-menu edits. Wayland used headless Weston. X11 used Xvfb and
Openbox. Both used Vulkan Lavapipe. [Published input hashes](inputs.sha256)
