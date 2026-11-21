# Correct the height of a saved Sketch Text fraction

The native regression passed on Linux X11 under Xvfb with Vulkan Lavapipe,
API v0.165.0, and Martensite v0.13.1. The tested tree
`d07ed30e1fec0d6c26d2788e7da416047f232649` matches application commit
`c4a2f0ac1700d3f146384c96918ecfd56dd042e1`. The corresponding Sketch Text feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

Open [Part1.opd](Part1.opd) and edit Sketch1 in Front view. The 47-dimension
sketch has a blue stacked Arial fraction at height 6.10 mm and width 50.
The Part SHA-256 is `714c3353e691330410b8c1f9c3f4da3090a130faba225b9aa7926b7d5744dbd1`.

The editor rejects zero and [shows a warning](height-fraction-warning.png).
After dismissal the previous preview returns. The next key is 6 followed
by Tab. The editor [displays the typed 6](height-fraction-typed-six.png),
keeps Arial, and previews the fraction at 6 mm. Cancel restores the saved
sketch and shows [Ready with 47 dimensions needed](height-fraction-cancel-at-ready.png).
The test checks that the complete authored sketch equals the saved sketch.

The font setup uses the same checksummed optional Arial archive as the
[font preview regression](../2026-11-21-sketch-text-arial-3907/README.md).
The native regression also passed with DejaVu Sans when Arial was absent.

```sh
XMODIFIERS=@im=none VK_DRIVER_FILES=/usr/share/vulkan/icd.d/lvp_icd.json \
  ./tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  sketch_text_rejects_zero_height_without_losing_the_last_valid_preview \
  -- --nocapture --test-threads=1
```

The source scenario remains open because its height assertion differs from
the typed value. See the [source discrepancy](../../scenario-defects/text-37.md).
