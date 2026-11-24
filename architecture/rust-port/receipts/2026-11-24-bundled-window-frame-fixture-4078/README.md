# Bundled window-frame fixture

Issue #4078. The standalone window-frame smoke test at `2d55417ce` failed
before opening a window because its scenario checkout did not contain
`fixtures/cad/main-window-frame-start.opd`. The repository already ships a
saved Part for standalone tests. The smoke test and taskbar projection test
now open disposable copies of that Part. Scenario steps continue to use
the file from their selected checkout.

The bundled Part had SHA-256
`0021108d0d19892ae7662daf7004e6d101873c17cfba3218c1fdae57d61c85b8`.
The standalone smoke test passed with `OBK_BDD_ROOT` pointing to a nonexistent
directory. It verified the visible solid, accepted a sketch edit on the
copy, checked dirty state, and compared the original fixture bytes.

The taskbar projection test also passed with the same nonexistent scenario
root. It exercised shrink, maximize, restore and minimize, checked the 40
and 25 dimension captions, and restored the application from tint2. Both
tests ran on Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. Strict Clippy
passed for all desktop targets.
