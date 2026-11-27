# Pinned Viewport Zoom baseline

All seven scenarios and 74 steps passed with API v0.161.0. The application
was at `da2393b17f63f9adb92ca4301fc8c422cc0c93d4` with local changes to
unrelated test code. The source checkout was clean before and after the
run at `b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`cb07444a13797a6d98fac5b8f26463343a1cbff5ed4781c359290bd6e4856e9c`.
The live source has since changed.

The wheel used X11 input. Other actions used the scripted desktop frame
loop. The host prepared the initial solid and view. The steps checked the
visible outline, camera orientation, selected face, menu state, and
cancellation. The saved captures show both Zoom Selected cases.

```sh
OBK_BDD_FEATURE=viewport-zoom XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```

Set `VK_DRIVER_FILES` to the installed Lavapipe manifest to reproduce this
Linux run. The Go wheel and modal tests provide partial reference evidence
in [the Go baseline](../2026-11-14-go-zoom-baseline/README.md). This is R0
baseline evidence. Strict acceptance requires bounded build and runtime inputs
under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
