# Current native File Menu

All eight current scenarios passed individually, with 67 steps. The
cases cover the File Menu over a saved Part, an empty Part, a Drawing,
and Home. Escape and an outside click preserved the active document.
The recent-documents check found the document-type symbol, name, and
pin on the row created by opening the published Part. Save As and New
showed their submenu entries.

| Scenario | Steps | Native result |
| --- | ---: | --- |
| Open the file menu while a part is active | 10 | [Cucumber](part-baseline/cucumber.json) |
| Dismiss the file menu with Escape | 7 | [Cucumber](escape/cucumber.json) |
| Dismiss the file menu by clicking outside | 7 | [Cucumber](outside/cucumber.json) |
| Open and dismiss the file menu in a drawing | 8 | [Cucumber](drawing/cucumber.json) |
| Inspect controls in the recent-documents region | 12 | [Cucumber](recent-final/cucumber.json) |
| Inspect the Save As submenu for a saved part | 6 | [Cucumber](save-as/cucumber.json) |
| Open the New submenu from the File menu | 10 | [Cucumber](new/cucumber.json) |
| Inspect the File menu from Home | 7 | [Cucumber](home-final/cucumber.json) |

The builds used base revision `81e1e4a614ae99f24f5d1e18677b92dfae030606`,
the scenario bindings available for each run, and existing local GUI
filesystem, opening, and saving edits. The native window used scripted
input on Linux X11, Xvfb, Openbox, and Vulkan Lavapipe. The source and
fixture hashes matched the shared source and isolated copy after
the runs. Strict build and runtime provenance remains open under #3978.
See the [combined report](cucumber.json) and [input hashes](current-inputs.sha256).
