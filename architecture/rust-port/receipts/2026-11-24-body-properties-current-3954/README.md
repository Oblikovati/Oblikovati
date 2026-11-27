# Current native Body Properties

All six scenarios passed individually, with 69 steps, using disposable
copies of the published Part. The physical fields showed N/A until
Update, then displayed the expected mass, area, volume, and center of
gravity for the fixture's 40 × 30 × 20 mm solid. The values persisted
when the dialog reopened in the same session. The appearance cases
verified applying and restoring Black and discarding pending changes
with Cancel, Escape, or the title-bar Close control.

| Scenario | Steps | Native result |
| --- | ---: | --- |
| Update unavailable physical properties for a solid body | 17 | [Cucumber](update/cucumber.json) |
| Cancel a body appearance selection | 13 | [Cucumber](cancel/cucumber.json) |
| Apply and restore a body appearance | 13 | [Cucumber](restore/cucumber.json) |
| Dismiss body properties with Escape | 6 | [Cucumber](dismiss/cucumber.json) |
| Discard a pending appearance with Escape | 10 | [Cucumber](escape/cucumber.json) |
| Discard a pending appearance using the title-bar close control | 10 | [Cucumber](title/cucumber.json) |

The build used base revision `e18df6cb6802a4eec8b2bf3fe73500fd830ce54d`,
the current fixture bindings, and existing local GUI filesystem,
opening, and saving edits. The native window used scripted input on
Linux X11, Xvfb, Openbox, and Vulkan Lavapipe. The feature and fixture
hashes matched the shared source and isolated copy before and after
the runs. Strict build and runtime provenance remains open under #3978.
See the [combined report](cucumber.json) and [input hashes](current-inputs.sha256).
