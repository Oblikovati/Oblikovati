# Typed Delete Body checkpoint replay

Issue #4079. A Part with two independent prism bodies retains its current
checkpoints. The regression deletes the first body, verifies the survivor's
stable key, and deletes the last body. At `7178b34b0`, the two typed calls
advanced the full replay count from one to three.

Delete Body now uses the document's checkpoint edit path. It replays the
deletion tail and publishes the accepted generation before listing the
remaining bodies. The original refresh path still handles edits without a
safe current prefix, including staged transaction edits.

The focused head suite passed eight tests. The full replay count stayed at
one through both deletions, and the final generation retained the successful
Delete Body record with zero remaining bodies. The automation suite passed
77 tests. Clippy passed for all automation and head targets with warnings
treated as errors.

The native Model Browser scenario **Hide and show a solid body from the
browser** passed all 13 steps on both builds under Xvfb and Openbox. This
scenario exercises the browser against the current body list. The focused
head regression exercises the typed deletion itself.

- Scenario source: `b576445b98cffd686dcc05aa8c017de2a591c8c8` (clean checkout).
- Feature SHA-256: `603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
- Reports: [baseline](before/cucumber.json) and [updated build](pinned/cucumber.json).
