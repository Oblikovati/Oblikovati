# Parallel GUI context tests

Issue [#3878](https://github.com/Oblikovati/Oblikovati/issues/3878) covers the
process-global Dear ImGui context shared by tests in one process. The baseline
run at `3c94ce536` used eight test threads and failed 56 tests. The narrowed
12-test icon run failed 11 tests, with occupied-context errors and a native
context-creation assertion.

The GUI unit-test build now uses one reentrant, thread-bound lease for the
context lifetime. The wrapper also covers direct context fixtures. Production
builds retain the existing occupancy checks without the test lease. The full
gate still runs its native and GUI tests serially because native window
captures share a display.

Two eight-thread library runs passed 493 tests each. A separate focused run
passed the concurrent-owner, nested-owner refusal, and panic-cleanup checks.
The production context-ownership integration test passed. Strict Clippy passed
for all GUI targets. The test logs in this folder include both passing runs and
the baseline failure; these are test evidence, not an application BDD receipt.

A final eight-thread run with the added nested-owner and panic-cleanup tests
passed 495 tests. The source guard passed two tests and permits raw constructors
only in the production engine, the leased unit-test helper, and the separate
process-isolated production ownership probe. It detects direct `Context::create`
and `igCreateContext` calls; it does not claim to detect renamed aliases.
