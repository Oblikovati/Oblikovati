# Maximize and restore within the desktop work area

The published button scenario passed all 11 steps on Wayland at
1920 × 1080 and at 3840 × 2160 with 2× scaling. It also passed all 11
steps on X11. The Wayland test measured the Weston output and its
top panel, then checked that the maximized application occupied
the remaining rectangle. Restore returned it to the original desktop
position and size. The active Part stayed visible and the ribbon and
viewport fit inside the enlarged drawable.
[Wayland report](wayland.json) · [4K Wayland report](wayland-4k.json) · [X11 report](x11.json)

Both Wayland runs used a single-output private Weston desktop with the
scene-graph probe. X11 used Xvfb, Openbox, and tint2. Each run used Vulkan
Lavapipe and the same published fixture.
[Compositor geometry](geometry.txt) · [Input hashes](inputs.sha256)

The baseline `9b6482647` reached Maximize on Wayland, then tried to read
X11 work-area properties. The passing build includes the new test
branch and the existing local GUI filesystem edits.
[Baseline report](baseline.json) · [Earlier native control checks](../2026-11-26-wayland-maximize-restore-4063/README.md)
