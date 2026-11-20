# Reopen Sketch Text by double-clicking

The Linux native regression created bold `ab` at 3.05 mm, then used Ctrl+Z
and Ctrl+Y to remove and restore the text. A single click on the glyph
and a double-click on empty canvas left Format Text closed. A double-click
on the glyph reopened the retained text and formatting. Cancel preserved
the authored text and solver count. The test used X11 keyboard and pointer
input with Xvfb and Vulkan Lavapipe.

```sh
XMODIFIERS=@im=none tools/with-xvfb.sh cargo test --locked --offline \
  -p oblikovati-head --lib \
  native_double_click_reopens_redone_bold_text_and_cancel_preserves_it \
  -- --nocapture --test-threads=1
```

The native regression uses DejaVu Sans. The pinned `text-14` scenario
requires Tahoma. The runner checks for the named family when it creates
the fixture. This machine does not have Tahoma. The full pinned scenario
remains open. The feature SHA-256 is
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.
