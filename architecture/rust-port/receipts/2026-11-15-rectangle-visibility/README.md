# First-part rectangle visibility

The native first-part journey now projects the committed sketch endpoints and
checks contrast along all four edges inside the document area. It rejects the
probe when a GUI window intersects it. The capture was inspected: the sketch
rectangle is visible in the document viewport before browser re-entry.

The oracle tests passed six cases, including a missing edge, a lone matching UI
pixel, occlusion, invalid bounds, and a gap. The full native journey passed
under Xvfb and lavapipe. It temporarily hides the actual sketch through the
typed API, confirms the oracle fails, aborts the transaction, and verifies the
undo and redo state and the complete history entry list are unchanged. The
restored rectangle, camera, browser re-entry, extrusion, edit, and save/reopen
checks then pass. Strict Clippy passed for all head targets.

This run injects input into the native frame loop; it does not establish
operating-system keyboard or mouse delivery. The screenshot and logs are test
evidence, not a scenario acceptance receipt. Refs [#3969](https://github.com/Oblikovati/Oblikovati/issues/3969).
