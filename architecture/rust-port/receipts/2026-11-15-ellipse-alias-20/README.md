# Ellipse alias scenario

The `ellipse-20` scenario passed all eight steps against pinned source snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. The application base was
`6e4a49ff1` with the two wording bindings and the axis-editor assertion in
this change. The checkout was unchanged during the run. The feature SHA-256
was `af53af88f7a32b441d76154fd75aeb36e68d4e673c47d21ecefa5be14cef5a6f`.

The intermediate capture shows `Command alias input: 6`, the previous sketch
geometry, and no axis editor. The final capture shows the state after Escape.
The test also compares retained geometry. It uses Xvfb, Lavapipe, and
frame-loop input; it does not prove OS-level mouse or keyboard input.

This focused pass is not verified acceptance: native build provenance remains
open in issue #3978. The earlier unbound result is retained in
`../2026-11-15-circle-midpoint-3996/ellipse-20.log`.
