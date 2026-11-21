# Face properties

The selected-face context menu opens a dockable Face Properties window. It
shows the selected reference and an analytic area in square millimetres. The
owner validates the current document and exact face before querying the area.
A changed document or model generation retires the window state (#4053).

The owner test checks the 40 mm by 30 mm fixture face against 1,200 mm² and
checks stale-state retirement. A native X11 test opens the window from the
face menu and closes it through its title-bar X. Its captured frame is in
`receipts/2026-11-17-face-properties-4053/`. This is diagnostic evidence;
the separate standard menu, face export, and Select Other remain open.
