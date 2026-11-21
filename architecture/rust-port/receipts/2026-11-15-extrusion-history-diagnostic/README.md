# Extrusion history diagnostic

- Application revision: `8ce0de2b6`.
- Scenario revision: `48987af7b5cbb06515bc6f30385f508382f34064`.
- Environment: Linux, Xvfb and Vulkan Lavapipe.
- Result: three of four scenario steps passed. The Measure step was undefined.
  This is a diagnostic, not a passing scenario receipt.
- The native test exercised the frame-loop inputs for OK, Undo and Redo. It
  checked the feature count, browser rows and shaded solid. The screenshot was
  reviewed and shows the restored solid and `Extrusion1` row.
- The separate owner tests checked analytic edge lengths after edit, history
  and reopen, and rejected wrong and deleted references. They did not select
  edges in the viewport.
- Frame-loop input does not establish OS-level keyboard or mouse delivery.
  Viewport edge selection and the Measure result are tracked in #3967.
