# Home workspace chrome

Home showed Get Started and Tools with the ribbon display controls
intact. The application and native window titles reflected Home, and
the status bar cleared the Part's zoom telemetry while its docks were
suspended. Returning to the Part restored the browser, tabs, title, and
zoom. The empty startup Recent list kept its column headings.

Both Home scenarios and all four Ribbon Display Home transition
scenarios passed on Wayland and X11. They covered full and tabs-only
modes, document settings availability, and dismissal of temporary
commands. The full Wayland runs passed 36 of 37 Home Recent Documents
scenarios (403 steps) and 48 of 52 Ribbon Display scenarios (551 steps).
The remaining Home scenario stops at the missing resize setup step.

The native builds used `bc38383ff`, the workspace changes in
`e604c1905`, and existing local GUI filesystem edits. The full reports
include reruns of the six focused cases after the tabs-only Tools
selection fix. The GUI library passed 840 tests with one ignored before
that final adjustment. The 24 focused Home GUI tests passed afterward.
Inputs matched their recorded hashes after the runs. Strict build and
runtime provenance remains open under #3978.

- [Home Wayland report](home-wayland.json) and [arguments](home-wayland-arguments.json)
- [Home X11 cases](home-x11.json) and [arguments](home-x11-arguments.json)
- [Ribbon Wayland report](ribbon-wayland.json) and [arguments](ribbon-wayland-arguments.json)
- [Ribbon X11 cases](ribbon-x11.json) and [arguments](ribbon-x11-arguments.json)
- [Home inputs](home-inputs.sha256) and [Ribbon inputs](ribbon-inputs.sha256)
