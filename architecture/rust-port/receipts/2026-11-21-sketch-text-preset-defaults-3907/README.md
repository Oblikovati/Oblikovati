# New Sketch Text preset defaults

The native keyboard and mouse regression passed on Linux with X11, Xvfb,
and Vulkan Lavapipe. It used the published API v0.163.0 and the application's
pinned Martensite revision.

Use Current Preset restored saved `g` at width 100. Last Used restored the
accepted blue stacked fraction at width 50. The selector showed a clock and
`*g`, and the Settings menu showed Last Used checked. Choosing No Preset
returned to a fresh, empty editor at height 3.05 mm and width 100. The
editor used the application's inherited font. Preset `g` remained in
the list, and the accepted sketch was unchanged.

[Last Used selector and fraction preview](last-used-asterisk-and-clock.png)

The native regression is
`sketch_text_new_placements_distinguish_named_and_last_used_defaults`.
It covers the preset settings and the switch between the saved and accepted
states. Pinned `text-32` still needs a fixture that establishes Tahoma as
the fresh editor default. This record gives no passing result for the
complete pinned scenario.
