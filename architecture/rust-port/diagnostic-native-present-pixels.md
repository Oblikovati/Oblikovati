# Native presentation pixel mismatch — Linux acceptance open

The native history test fails on this host with Martensite v0.2.1,
revision `193572dc3665598c4948777c87375bad7dc8114a`, RADV STRIX_HALO,
XWayland display `:1`, and Vulkan validation layer 1.4.321.

## Reproduction

```sh
VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers" \
  CARGO_NET_OFFLINE=true cargo test --locked -p oblikovati-head --lib \
  native_history_keys_restore_prism_pixels_without_changing_the_camera \
  -- --nocapture
```

This command fails its existing three-second exact native pixel comparison.
It also fails at clean pre-GUI commit
`d56ac5497` in a detached diagnostic worktree. The worktree was removed after
the check. The failure predates the GUI packet producer; this does not establish
the underlying cause or excuse hardware acceptance.

## Observed boundary

Temporary probes compared the independent X11 drawable readback with kernel GPU
capture. Both are 1440 by 900. GPU capture has the new history state and status
text; native readback retains the preceding edited prism without the text.
One capture had 24,300 green model pixels and 111 white text pixels; native
readback had 36,450 green pixels and no white pixels. The model also differs,
so this is not only a glyph mismatch.

The event trace contains mapping, exposure, resize, and focus, with no unexpected
key command. A debugger probe after `queue_present` reports successful
`Ok(false)` results. It did not observe an out-of-date swapchain error.
Forcing Mesa FIFO presentation through the environment still failed.

The same test passes with Mesa's software Vulkan driver on both the current
XWayland display and an isolated Xvfb display, with validation still enabled.
The AMD device cannot present through that Xvfb server because it lacks DRI3.
These comparisons narrow the failure to the hardware presentation path on this
host. They do not distinguish RADV, XWayland, kernel presentation, or their
interaction. No root cause is claimed.

## Acceptance and ownership

Keep the native pixel oracle, validation, and timeout unchanged. Software-driver
results are additional evidence; they do not certify the AMD path. The full
hardware gate remains open until the original reproduction passes.
All temporary source probes were removed. The released kernel checkout and the
kernel owner's active source were not changed. Any required kernel fix must be
tested and released by that owner, then consumed by the application.
