# Ellipse measurement blocked by conic extrema

The corrected `ellipse-05` step was run against source snapshot
`e76cb749ed43c73a91c24854057da7c890041f51` and application base `a00d59dbb`
with the binding and visible-pixel assertion in this change. The feature
SHA-256 was `af53af88f7a32b441d76154fd75aeb36e68d4e673c47d21ecefa5be14cef5a6f`.
The scenario passed its first four steps and failed at ellipse selection.
The screenshot shows the ellipse and the open, empty Measure panel.

A separate instrumented repeat of that same setup recorded the click at
`[1003, 318]` and cursor `[2.9111111111111105, -6.288888888888888]`. The
analytic query returned a boundary witness at `[1004, 304]`, outside the
12-pixel selection radius. The ellipse and cursor are the existing independent
OCCT parity regression in `crates/martensite/tests/planar_pick.rs`, tracked by
[Martensite #406](https://github.com/Oblikovati/Martensite/issues/406). The
instrumentation was removed after the run; its output is in `witness.log.gz`.

The run used Xvfb, Lavapipe, and frame-loop mouse and keyboard input. It is a
failed focused diagnostic, not verified acceptance. It does not show that
all ellipse measurement paths fail. Native build provenance remains open in
issue #3978. No kernel workaround was added.
