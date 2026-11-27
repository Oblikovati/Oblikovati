# Viewport Navigation Wheel baseline

The pinned source `b576445b98cffd686dcc05aa8c017de2a591c8c8`
describes an active Part with a solid and gives no file path.
The published feature uses the same steps and removed the
`@needs_fixture` tags. Its fixture notes name `main-window-frame-start.opd`.

The pinned feature SHA-256 is
`9793d6a9ad9655ef4412295075dfbf9f6f5a5f97566e5564f2ca9f5a9c68f62a`.
The published feature SHA-256 is
`8b6164be54d7a3bd307b4de24d7750079ce4d23b9ada2e6fec7a3b83149e2cfa`.
The pinned fixture SHA-256 is
`0021108d0d19892ae7662daf7004e6d101873c17cfba3218c1fdae57d61c85b8`.
The published fixture SHA-256 is
`cac3df2175ff0f191abda7152700a4542903f2281d4a1668f09315d48dc66664`.
The YAML content is identical. The repository copy uses LF on all lines.

The original pinned run failed to open an active Part because the
fixture path did not exist in that snapshot. The fixture selection
regression checks the repository copy, a fixture present in the source,
and the path from an explicitly supplied source. Every scenario opens
a disposable copy through the native dialog.

The [Go reference](../2026-11-14-go-wheel-baseline/README.md) covers
toggling and cancelling the wheel, wedge placement, and tool dispatch.
The Rust benchmark also checks wheel layouts, popup menus, preferences,
camera changes and the repeat command.

The [pinned run](pinned/cucumber.json) and the [published run](live/cucumber.json)
each passed 12 scenarios and 143 steps at
`9fe930dabc0852c78922279c2c856ced0293051e`. They used Linux X11 input with
Xvfb, Openbox and Vulkan Lavapipe. The local [fixture regression](fixture-regression.log.gz)
also passed. The first pinned case started before the fixture's last line
ending changed. A separate [rerun with the normalized file](normalized-fixture-first-case.json)
passed all 15 steps.

The [full wheel](full-wheel.png) shows the segmented controls. The
[home view](home-view.png) shows the same Part after the wheel closes.
These runs remain historical evidence while
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978) is open.
