# File Menu native baseline

All eight pinned scenarios and 59 steps passed on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe. The application was
`61dddbd951df3baec11cdac62f358c07cb370b97` with API v0.165.0 and
Martensite v0.13.1. The pinned source was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The pinned feature SHA-256
was `fd33924caea15cf97ba75ad3600c7244cf17592b6bdf26bdc9882f7cae71fb91`.

The [pinned report](cucumber.json) records menu presentation in Part,
Drawing, and Home, Escape and outside-click dismissal, recent-document
controls, and the New and Save As submenus. The [Save As capture](save-as-submenu.png)
shows the four choices. The [X11 input test](native-input.log) clicked File
and dismissed it with Escape and an outside click. Both actions kept the
Part active. The [open menu capture](native-menu.png) shows the menu above
the active Part.

The current feature SHA-256 was
`e8461347a263ea25b25ebbc31e8431f3cbf0573797571d13838ee725964d5ab0`.
The [current report](live/cucumber.json) records six passes and two undefined
fixture steps. The Save As and Home scenarios name
`fixtures/cad/planar-sketch-profile-kit.ipt`. The scenario owner's
`BLOCKERS.md` identifies the existing native `planar-sketch-empty-xy.opd`
as a replacement for these inspection-only cases.

The [Go baseline](../2026-11-14-go-file-menu-baseline/README.md) records
the related state and dispatch tests. The native results are historical
evidence while [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978)
remains open.
