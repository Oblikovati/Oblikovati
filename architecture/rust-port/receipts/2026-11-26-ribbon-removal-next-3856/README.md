# Continue removing staged ribbon commands in order

The native test stages four available commands and uses Shift+Up from
the second row to select the first two. Remove clears those rows from
the target and selects the first surviving command. Two more
Remove clicks clear the target in order without reselecting. The
target keeps its pending marker until the test cancels the draft.

The baseline on Wayland and X11 skipped the first surviving row after
removing the range. The fix accounts for the removed rows above the
original selection when choosing the next row.

The published **Remove successive staged commands without reselecting**
scenario still needs 3D PDF. Its [baseline](published-baseline.json)
stopped at the 3D PDF setup step. The native regression uses the
current catalog and checks the rendered selection at each removal.

The build used `e7298d429` with the removal changes and existing local
filesystem edits. Wayland used headless Weston. X11 used Xvfb and
Openbox. Both used Vulkan Lavapipe. [Test results](test-results.txt)
and [published input hashes](inputs.sha256)
