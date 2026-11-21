# Restore new Sketch Text defaults

Pinned `text-32` passed all ten steps on Linux X11 under Xvfb with Vulkan
Lavapipe, API v0.165.0, and Martensite v0.13.1. The tested tree
`3e1132b7565aa7b0de3276117697406d0ff85bcc` matches application commit
`a9519616f21c7f601219091cb2ead8e21db9d079`. The clean scenario snapshot was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The test creates named preset `g` with the stacked fraction at 100 percent
width, then accepts a version at 50 percent. Use Current Preset opens the
saved version. The Last Used setting opens the accepted version and shows
the [clock and `*g` selector](last-used-clock.png). The setting remains
checked when its menu is reopened.

Restoring No Preset opens an empty editor with
[Tahoma, 3.05 mm, a purple swatch, and 100 percent width](no-preset-tahoma-and-purple.png).
The named preset remains listed and the authored sketch is unchanged.
A separate native run without Tahoma checked the System default fallback.

The isolated Tahoma run used the same Fontconfig setup and CAB checksum
as the [retained Tahoma fixture](../2026-11-21-sketch-text-tahoma-3907/README.md).
To repeat it, set `FONTCONFIG_FILE` to a configuration that can read Tahoma.

```sh
XMODIFIERS=@im=none VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-32 \
  ./tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head \
  --lib sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The Go Sketch Text and glyph tests passed on
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. These reports supply R0
baseline evidence. Strict verification remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
