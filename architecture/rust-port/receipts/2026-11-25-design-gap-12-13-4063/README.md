# Document tabs and viewport telemetry

Native Linux verification used Xvfb, Openbox, and Vulkan Lavapipe. The build
included the working ribbon and browser appearance changes and existing local
file-system and dialog test edits.

- The tab strip measures 26 px, and the status bar measures 24 px. The solid
  workspace displays the document’s units, model-plane pointer coordinates,
  and zoom. Halving the camera scale changes the status from 100% to 200%.
  [Captured frame](document-strip-and-status-telemetry.png)
- The published **Inspect a document tab's context menu** scenario passed all
  six steps, including the final Properties row and keyboard shortcuts.
