# Drag to place linked Sketch Text

Pinned `text-16` passed all seven steps on Linux X11 under Xvfb with
Vulkan Lavapipe, API v0.165.0, and Martensite v0.13.1. The tested tree
`9d7bcdaa2bfe600746eaf7c1a08516d1b642f352` matches application commit
`d594368933b05c7b7569f10bf99128c3ebebc2f4`. The clean scenario snapshot was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

Open [Part6.opd](Part6.opd) and edit Sketch2 in Front view. The fixture has
a 10 mm model dimension named `d0` and starts with 36 dimensions needed.
The window is 1440 × 1000 client pixels so both recorded drag coordinates
are inside the viewport. The Part SHA-256 was
`0b5e63811c755c6e62801561ef026356433b1d2d65cd45ce85bfd1711911eab2`.

The test drags from (1050, 800) to (1250, 900). Format Text opens empty
after release. Choosing `d0` and using Insert displays the
[shaded model field](linked-d0-drag-editor.png) at `10.000`. After OK and
Escape, the [viewport](linked-d0-drag-at-ready.png) shows that value at
the press point. The sketch stores the field reference and has 38
dimensions needed.

The focused run used the verified pinned checkout as `OBK_BDD_ROOT`
and a new `OBK_BDD_REPORTS` directory.

```sh
XMODIFIERS=@im=none VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-16 \
  ./tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head \
  --lib sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The shared filename fixture regression `text-34` passed all ten steps.
Go Sketch Text and glyph tests passed on
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The report supplies R0 baseline
evidence. Strict verification remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
