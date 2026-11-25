# Model Browser and viewport annotations

Native verification used Xvfb, Openbox, and Vulkan Lavapipe. The builds included
the existing local file-system and dialog test edits.

- The active asset tab uses the brand underline and primary text. The other
  tab uses muted text. [Native frame](Appearance-Browser.png)
- **Search the Model tree from another active dock tab** passed all 11 steps.
  The dock search control activated Model, focused the search field, and
  retained the query when called again from Log.
- **Filter the model browser by a feature name** passed all 12 steps.
  Clearing and closing the integrated field restored the hierarchy.
- **Select a solid feature in the browser** passed all 10 steps with the
  Studio teal outline. The published scenario still uses the earlier
  color name.
- The dimension rendering regression passed at both framebuffer scales.
  Linear annotations draw extension lines from the measured endpoints
  and place the unit-bearing caption clear of the measurement line.
  [Model Browser and dimensions](model-browser-and-dimensions.png)
- The published hidden-caption expansion scenario passed all 15 steps
  after the disclosure control was aligned with the centered group caption.
- The published Home and sketch keyboard scenario passed all nine steps
  using the fixture’s full path in the native Open dialog.

- The native Home grid accepted typing, Search, and clear through its joined
  search field. [Home grid](x11-home-grid-restored.png)

The repository-wide gate reports 2,183 scenarios missing strict R0 evidence.
