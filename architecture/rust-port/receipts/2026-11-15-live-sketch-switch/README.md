<!-- SPDX-License-Identifier: GPL-2.0-only -->

# OS-input sketch switch check

On 2026-11-15, XTest sent mouse events to the standalone desktop on Linux Xvfb
with Vulkan Lavapipe. The application was built at `d78b9c53f`; the root README
had an unrelated local edit. The executable SHA-256 was
`c90ad9369eb11681b3918d9ac674505884067c21e34b5c2461d9051437b3454f` and the
MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.

The user input created a part, selected XY, started a sketch, drew a rectangle,
and stopped the drawing tool. The Home tab displayed the recent-documents
workspace while the part remained open. Selecting the part tab restored the
sketch. MCPBridge captured the window before and after the switch. After the
pointer left the tab, the two PNG files had the same SHA-256:
`b69e65ffb97fac17992cebdc1f64a847580d86e868fab7a361b189506c85e26a`.
The retained capture shows the rectangle, sketch controls, browser and four
remaining degrees of freedom. The Home capture is not retained because it
shows the local recent-documents list.

The click coordinates were New Part `(100,120)`, XY Plane `(120,524)`, Start
2D Sketch `(100,125)`, Rectangle `(100,120)`, rectangle corners `(860,500)`
and `(1100,650)`, Home tab `(510,266)`, and Part tab `(650,266)`. Escape ended
the rectangle tool before the switch.

This is a manual OS-input observation, not a BDD acceptance receipt. It uses
the Home tab; the Switch menu is covered by the separate frame-loop diagnostic.
The build inputs remain unbounded (#3978). The host and Xvfb were closed after
the check. `input.py.txt` and `run.sh.txt` retain the test setup; the pointer
coordinates depend on the 1900 × 1100 window size.
