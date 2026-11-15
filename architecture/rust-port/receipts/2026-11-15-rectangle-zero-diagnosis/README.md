# Zero-size rectangle diagnosis

At application commit `f104b9fd6c109b47e3a3507dac3cb2cc09efa7f1`, the
unmodified pinned scenario 10 failed at the expected transient count of 24.
The displayed count was 22. The scenario stopped at that assertion.

A separate instrumented diagnostic replaced that assertion to observe the
state through Escape. It is not a passing scenario run. On the constructed
fixture, geometry, model DOF (22), document generation (7), and undo and redo
labels remained unchanged before Return, after Return, and after Escape. The
fixture has five free lines and one free point; it is not the unspecified
recorded sketch. The Go rectangle recipe attempts a four-edge commit but does
not establish the recording's transient count or fixture.

The source scenario's existing `BLOCKERS.md` asks for the recorded geometry,
first corner, automatic constraints and intermediate state. No product defect
was established from this substitute fixture. Keep
[issue #3953](https://github.com/Oblikovati/Oblikovati/issues/3953) open until
the premise can be checked. The two compressed logs distinguish the unmodified
failure from the instrumented continuation; the latter contains temporary test
instrumentation and is diagnostic only.
