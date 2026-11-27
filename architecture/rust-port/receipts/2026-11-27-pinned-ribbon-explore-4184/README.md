# Pinned Explore ribbon panel visibility

The scenario used clean pinned source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and application code
`6e6a6af4a00c00e15ad15e6545729f636cf97612`. All 16 steps passed on
[X11](x11.json) and [Wayland](wayland.json). The matching published case
passed 17 steps on each backend with its named Part fixture.

The scenario checks Explore's actual submenu state and rendered caption,
and measures the Work Features heading before hiding Explore, after the
reflow, and after restoring Explore.
