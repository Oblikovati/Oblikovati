# Maximize and restore from the Wayland application bar

The native Wayland test opens the saved Main Window Frame Part and
clicks the rendered Maximize and Restore control. It checks the
compositor's maximized state, growth on both axes, the Restore tooltip,
and the exact original drawable size after returning to Maximize. The
ribbon and viewport fit inside the surface at each size, and the same
Part stays active with its solid visible.

The Wayland protocol trace records both requests and the compositor's
configure responses. The test uses physical framebuffer coordinates
for the ribbon and control bounds. It passed at 1920 × 1080 and on a
3840 × 2160 output at 2× scale. Weston ran its desktop shell with a
reserved top panel. The published X11 case passed all 11 steps,
including the global work-area and placement checks.
[Wayland trace and test results](test-results.txt) · [Published X11 case](published-x11.json)

The baseline `808e37d6a` attempted to connect to X11 when the published
scenario clicked Maximize under Wayland. The passing build used the
backend-aware native test input and existing local GUI filesystem edits.
The published fixture and source matched before and after the runs.
[Baseline Wayland case](baseline-wayland.json) · [Input hashes](inputs.sha256)

Run on a Wayland desktop compositor with room to maximize:

```sh
env -u DISPLAY OBK_BDD_ROOT=/path/to/MCAD-Scenarios \
  cargo test --locked -p oblikovati-head --lib \
  wayland_frame_maximizes_and_restores_the_saved_part -- --ignored --test-threads=1
```
