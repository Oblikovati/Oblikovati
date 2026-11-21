# Viewport Orbit native baseline

All ten pinned cases and 94 steps passed on Linux X11 with Xvfb, Openbox,
and Vulkan Lavapipe. Nine scenario declarations include one outline with
two examples. The application commit was
`4055c7951575a4712ac900705f20c2dcd6551637` with API v0.165.0 and
Martensite v0.13.1. The pinned source was
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and the pinned feature SHA-256
was `d0e82e1671dc51da3c8eafa7d25101dc66713552639adf9dec041fac6d05eeb6`.

The [Cucumber report](cucumber.json) covers free and constrained orbit,
the rotation guide, incremental rotation with preview, Reset, Cancel and
OK, all three opposite-direction pairs, and exit through Done, Escape,
Pan, and context-menu OK. The [accepted increment](accepted-increment.png)
shows the rotated block after OK. The [opposite-button example](opposite-button-example.png)
shows the restored orientation after the second outline example.

The [Go reference](../2026-11-14-go-orbit-baseline/README.md) covers the
camera input and constrained navigation calculations. The native report
checks the complete desktop dialogs and status changes. The results are
historical evidence while [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978)
remains open.

## Current scenarios with a native Part

The current feature SHA-256 was
`c2596fe0bc5affa50da21deece0cae5222032fc64c3d9db816a691ced84bf293`.
It names an Inventor Part. A scratch copy changed only the nine fixture
paths from `body-properties-start.ipt` to `body-properties-start.opd`.
The resulting feature SHA-256 was
`de555d06b64ecaa717dcad029db43339c6fa048a1e80a7d2daa461dc4df4c837`.
The native fixture SHA-256 was
`50d6b58b9d2d48e778334b9bce4f1610c0401f88e3e37d304536120b6d099643`.
It is a copy of the [tested 40 × 25 × 20 mm Part](../../../../test-utilities/fixtures/document-saving/SaveAsFolder/ExistingSolidPart.opd).
The scenario source now has the native fixture and its `BLOCKERS.md`
lists the pending fixture-path changes.

Five cases initially reached the end of Rotate at Angle with a stale
file-open status. The [failing regression](native-fixture/status-red.log.gz)
reproduced the defect on the saved Part. At
`2405cedfdef9ddc30e0183810b3968f3d9b8b1b0`, both OK and Cancel
restored Ready, and the title-bar close restored the previous camera.
The [X11 test run](native-fixture/x11-fixed.log.gz) also clicked Orbit,
dragged the view, and pressed Escape. The guide disappeared and the
same Part stayed active. See the [orbit guide](native-fixture/free-orbit-guide.png)
and the [accepted rotation at Ready](native-fixture/increment-ready.png).

All ten cases and 94 steps passed in the
[native-fixture run](native-fixture/cucumber.json) after the fix. This
includes both opposite-button examples and all four ways to leave Orbit.
