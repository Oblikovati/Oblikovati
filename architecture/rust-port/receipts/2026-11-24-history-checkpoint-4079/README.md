# Checkpoint replay during typed Undo and Redo

Issue #4079. A one-body Part retains certified checkpoints and deletes its
only body. The typed Undo restores the body's stable key. Redo removes the
body and restores the successful deletion record. At `261b15890` both
geometry checks passed and the full replay count advanced from one to three.

Ordinary typed history navigation now prepares the target from a current
checkpoint under the same resolution policy. The session accepts the move
through its existing observer and history path, then checks the candidate
against the accepted generation before publishing it. An unavailable or
unsafe prefix leaves replay to the usual update. Tagged view and prepared
model-state history still use their existing navigation paths.

The full replay count stayed at one after Undo and Redo on the updated build.
The body deletion suite passed nine tests, the Part view history suite passed
four tests, and the automation library passed 77 tests. Clippy passed for all
automation and head targets with warnings treated as errors.
Two native extrusion tests also passed. They cover creation and history, and
a distance edit with cancel, history, stable edge keys, and reopening.
The full head owner suite reported the same nine failures before and after.
The baseline had 498 passes and the updated build had 499.

On Linux under Xvfb and Openbox, the pinned `extrusion-06` scenario passed
five of six steps before and after the change. Both runs reached the native
Undo and Redo commands and verified the restored browser rows and shaded
solid. The final step is undefined and requires the desktop solid-edge
Measure work tracked in [#3967](https://github.com/Oblikovati/Oblikovati/issues/3967).

- Scenario source: `b576445b98cffd686dcc05aa8c017de2a591c8c8` (clean checkout).
- Feature SHA-256: `ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
- Reports: [baseline](before/cucumber.json) and [updated build](pinned/cucumber.json).
