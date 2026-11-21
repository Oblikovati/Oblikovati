# Dock tab bar during a clipped frame

The resize regression aborted in Dear ImGui on the desktop test binary built
from the workspace at `04a58abd4`. A stack trace reached
`oblikovati_gui_draw_dock_tab_actions` and then
`ImGui::DockNodeBeginAmendTabBar`. At a 40-pixel window height, the dock tab
bar was current for the frame while the host set `SkipItems`. The app must
skip tab controls until the host can draw them.

Run the existing regression under Xvfb:

```sh
tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-gui \
  --test ribbon_click panel_overflow::a_clipped_child_is_closed_before_the_next_visible_frame \
  -- --exact --test-threads=1
```

The same test passed after adding the host visibility check. It still verifies
that the restored ribbon opens its panel and dispatches Rectangle. The wider
ribbon test binary also returned normally with 27 passing and 19 failing
assertions. Those failures remain part of [the gate issue][gate].

[gate]: https://github.com/Oblikovati/Oblikovati/issues/3860
