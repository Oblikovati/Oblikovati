# 4K X11 automatic UI sizing

Oblikovati opened the same Part in a 3840×2160 X11 drawable, with clean
application profiles and the same 144 DPI desktop setting. The application
used the full drawable for rendering in every capture.

| Mode | UI layout scale | Window controls height |
| --- | ---: | ---: |
| Previous default | 150% | 48 px |
| Automatic | 125% | 40 px |
| Explicit 100% | 150% | 48 px |

The automatic capture shows a shorter ribbon, a narrower model browser, and a
smaller ViewCube and navigation strip. The Part and its dimensions are visible.
The File menu opened with a native pointer click. The Application Options
dialog opened from that menu and showed the Automatic radio selected.
[Before](before.png) · [Automatic at 150% desktop scale](automatic.png) ·
[File menu](file-menu.png) · [Application Options](options.png)

The renderer used Mesa llvmpipe on an isolated Xvfb display for the repeatable
4K comparison. In the 1400×680 ribbon and browser region, the Automatic
capture was pixel-identical to the previous build with desktop DPI set to
120. A saved explicit 100% value was pixel-identical to the previous build
with desktop DPI at 144. The 96 DPI control run showed the 100% layout.

The maximized application was also inspected on the 3840×2560 COSMIC
desktop at 150% scale using the Radeon renderer. Its XWayland client measured
3840×2441 within the desktop work area. The titlebar, ribbon, browser,
ViewCube, document tabs, and status bar were visible.
[Maximized COSMIC XWayland capture](cosmic-xwayland.png)

The Linux DPI integration test covers the main and detached drawing windows.
It checks the native backing scale and the separate UI layout at startup and
during XSettings and X resource changes. It also verifies the explicit 100%
override and the return to Automatic. Persistence and GUI tests cover storing
the automatic choice, retaining the icon setting, and the existing scaling and
hit-region behavior.

A separate Radeon run at 1440×900 and 150% desktop scale exercised the final
Automatic build with X11 VSync disabled. The swapchain requested and used
immediate mode. It measured 104.2 FPS with 13.82 ms p95 in continuous
redraw and 57.5 FPS with 22.26 ms p95 during orbit.
[Performance traces](runs/)
