# Current native Document Tabs

All 39 scenarios and 343 steps passed against the refreshed published
source. The last seven cases opened the twelve saved overflow Parts,
the saved Presentation, and the published Revolve profile. They
verified revealing tabs, selecting an offscreen Part, switching numbered
windows, keeping their cameras and zoom levels independent, and closing
one window without closing the Part.

The earlier [sketch switch](../2026-11-25-document-tabs-saved-sketch-switch-3900/README.md),
[saved Parts](../2026-11-25-document-tabs-saved-parts-3900/README.md),
[unsaved close](../2026-11-25-document-tabs-unsaved-sketch-close-3900/README.md),
[drawing windows](../2026-11-25-document-tabs-drawing-and-parts-3900/README.md), and
[Home and preview cases](../2026-11-25-document-tabs-home-and-previews-3900/README.md)
retain their individual reports and build bases.

The last group used base revision `97006779ce7e974d9540934c21749c421d9b9af5`, the overflow and
numbered-window bindings, and existing local GUI filesystem, opening and
saving edits on Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe. The
shared [input hashes](current-inputs.sha256) matched the isolated copies.
Strict build and runtime provenance remains open under #3978. See the
[combined report](cucumber.json) and [scenario inventory](scenario-inventory.json).
