# Chamfer edit diagnostic (#4046)

On 2026-11-17 a temporary native-input regression passed all ten steps under
Xvfb and software Vulkan. It created a Chamfer, opened Edit Feature from the
browser, changed the distance with the keyboard, checked the private preview,
cancelled, reopened, and accepted the edit. The panel showed the stored edge
count and distance. The test source, diagnostic report, log, and captures are
included here. This is a targeted regression, not a verified benchmark case.

A desktop-owner test also checked invalid edits, unchanged history on refusal,
undo/redo, a named distance parameter, and save/reopen through its in-memory
file workspace. A Fillet edit regression, dockable-window guard, focused GUI
tests, lint, and documentation checks passed. The live scenario fixture and
build provenance remain open (#3978). The fixture sketch has four degrees of
freedom; the capture retains the dark-face issue #4044. No frame-rate result
was collected.
