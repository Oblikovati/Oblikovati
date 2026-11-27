<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Tab preview lifecycle check

On 2026-11-15 the native window regression
`hidden_tabs_survive_closing_a_window_and_show_the_survivor_preview` passed
under Linux Xvfb with Mesa software Vulkan. It opened two windows on a part,
hid the tab strip, closed the second window with Control+F4, restored the
strip, and hovered the surviving part tab. The capture shows the surviving
sketch workspace and its tab preview. The test sends keyboard and pointer
events through the GUI frame loop; it is not an OS-level input test.

Owner and GUI tests separately verify identity changes between frames, cache
retention for a stable surviving window, pending update ordering, and capture
of the surviving window after an identity change. Build provenance remains
unbounded (#3978), so this is diagnostic evidence rather than BDD acceptance.
