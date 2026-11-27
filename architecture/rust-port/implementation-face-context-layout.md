# Selected-face context layout

The selected-face popup shows a linear Properties and Home View region above
its radial command ring. The ring uses the Go marking-menu direction and
spacing rules with live text metrics. It centers on the pointer when the
document area permits, and the popup origin is clamped at the area's edges.
The selection toolbar hides while the popup is open and returns on dismissal.
The saved classic-menu setting retains its linear layout (#4055).

Native X11 diagnostics cover the menu layout, Escape, New Sketch, Measure,
Properties, Chamfer, customization, and the classic setting. The captured
frame is in `receipts/2026-11-17-face-context-layout-4055/`. The full scenario
remains open: face export, Select Other, and Select Tangencies are not yet
available. Measure and Look At remain linear rows below the ring, and the
live scenario is not verified acceptance evidence.
