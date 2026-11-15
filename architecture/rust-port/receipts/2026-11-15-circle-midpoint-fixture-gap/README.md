# Circle diameter edit: fixture gap

A focused `circle-center-point-11` run at application commit `a55445b6f` and
pinned source `e76cb749ed43c73a91c24854057da7c890041f51` reported 12 passing
steps. The screenshot shows the final circle and line after undo. The run used
Xvfb, Lavapipe, and injected frame-loop input.

This is not acceptance evidence. The fixture creates an ordinary line, while
the scenario requires a retained midpoint line. The final check counts lines
without checking a midpoint relation. Issue #3996 tracks the fixture and
assertion correction. Native build provenance also remains open in #3978.
