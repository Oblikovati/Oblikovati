# Native Navigation Wheel on Wayland

The Navigation Wheel runner sends clicks, pointer motion, right-clicks,
fixture path keystrokes, and Escape through the active Linux desktop.
Previously the wheel scenarios always targeted an X11 application window
and could not open the saved Part on Wayland. The focused Wayland trace
shows Enter, pointer motion, and mouse buttons delivered to Oblikovati.

All 12 published scenarios and 155 steps passed on Wayland and X11.
The Wayland report combines 12 isolated runs. The X11 report uses 11
cases from the first run and the compact-wheel rerun. The compact-wheel
pixel check now recognizes separators that
are lighter or darker than their neighbors. The saved block makes
darker separators visible in the compact wheel. That case passed all 17
steps after the correction. The separator unit test covers both contrast
directions, and the three independent Pan and Orbit tests also passed
on each backend after sharing the desktop input helpers.

The builds used application revision `710969946`, the test changes,
and existing local GUI filesystem edits. Wayland used nested Weston 13
with a 2048 × 1200 Xvfb host. X11 used Xvfb and Openbox. Both used
Vulkan Lavapipe. Published input hashes matched before and after the
runs. Strict build and runtime provenance remains open under #3978.

- [Wayland report](wayland.json) and [X11 report](x11.json)
- [Focused Wayland open and close](wayland-open-close.json)
- [Wayland input trace](wayland-input.txt)
- [Published input hashes](inputs.sha256) and [test results](test-results.txt)
