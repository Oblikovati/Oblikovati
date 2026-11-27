# Sketch Text filename and precision

Pinned `text-34` and `text-35` each passed all 10 steps on Linux X11
under Xvfb with Vulkan Lavapipe, API v0.165.0, and Martensite v0.13.1.
The staged tree `b6b3ec8ad9352b7dbb683f16b9b249a2b40c0c37` matches application commit
`2a380aa07bcc9e591f74d0e5882171f44cecf5b8`. The clean scenario snapshot was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

Open [Part6.opd](Part6.opd) and edit Sketch2 in Front view. The fixture
starts with 45 dimensions needed, model dimension `d0` at 10 mm, and
the existing linked Text displaying `10.000`. It has no Finish or
User parameters. The new Text starts at (975, 600) in the 1440 × 900
window. The Part SHA-256 is
`fec6b1abf063ae1d857bc432faf5d27a5ed79b4ee5fa96fb42cab793a82b12b5`.

The filename path shows Source Part6 without numeric precision. It keeps
the [shaded FILENAME field](text-34/filename-preview.png) while the canvas
displays `Part6.opd`. Finish has no source. User shows Part6 with no
parameters.

The model-parameter path scrolls the Precision list to see
[the eighth decimal place](text-35/model-field-precision-scrolled.png).
It selects two decimal places and inserts `d0` on a new line. At
[Ready](text-35/mixed-fields-at-ready.png) the viewport shows `Part6.opd`
and `10.00` in separate paragraphs, keeps `10.000` in the earlier Text,
and shows 47 dimensions needed.

The focused runs used the verified pinned checkout as `OBK_BDD_ROOT`
and a separate `OBK_BDD_REPORTS` directory for each case. For example:

```sh
XMODIFIERS=@im=none VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  OBK_BDD_FEATURE=text OBK_BDD_SCENARIO=text-35 \
  ./tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head \
  --lib sketch_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

The earlier save and reopen case `text-38` also passed all nine steps
on the same tree. Go Sketch Text and glyph tests passed on
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The reports supply R0
baseline evidence. Strict verification remains open under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
