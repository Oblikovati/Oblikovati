# Details filename fit

The Details view shortens long filenames before the modification date. Hovering
over a shortened name shows the complete filename. The row keeps the original
filename for selection and opening.

A native regression saved two parts whose filenames shorten to the same text.
It checked that both names end before the date column, selected each by its
full tooltip text, and confirmed that Open activated the intended part. The
regression passed on Wayland and X11. The published **Switch between details
and thumbnail file views** scenario passed all nine steps on each backend.
The Wayland capture showed the shortened names beside readable timestamps.
All 74 focused GUI tests passed.

[Wayland report](wayland.xml) · [X11 report](x11.xml) · [Published inputs](current-inputs.sha256)

The earlier full Wayland run passed the first 11 scenarios and 115 steps.
Weston then exited with a segmentation fault. The remaining 16 scenarios
failed in window setup, beginning with `ERROR_SURFACE_LOST_KHR`. The
[isolated follow-up](../2026-11-26-file-picker-context-menu-3889/README.md)
verified all 16 interrupted scenarios.

The focused build used base `6b09676c5c640f9890fbd07eab669a22f7af29fd`, this
change, and existing local GUI filesystem and test edits. Wayland used
headless Weston with Pixman. X11 used Xvfb with Openbox. Both used Vulkan
Lavapipe and frame-loop input.
