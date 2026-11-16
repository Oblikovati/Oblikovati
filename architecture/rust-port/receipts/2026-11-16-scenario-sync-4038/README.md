# Scenario source sync diagnostic

The source snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8` is a local
commit made from an unversioned directory, not an upstream commit. It excludes
`BLOCKERS.md` files and reference images. The bundle was cloned to a separate
directory and its 1,338 files were verified against the recorded hashes.

The original comparison found 10 added and 7 changed feature files. The source
changed during review; the final snapshot includes the reviewed follow-up edits.
The feature inventory matched once after capture. The final check found a later
change in the solid viewport interaction feature; it remains for the next sync.
Markdown and SVG files were copied to the snapshot, but live drift checks cover
feature files only.

The parser reports 2,760 scenario declarations, including 10 in `sketch3d`.
The latter have no native runner and are tagged capture-only. All new
declarations have classifications; source changes leave 124 old classifications
stale. The ledger check fails on those stale rows, as intended. No scenario
pass is claimed by this sync. Earlier receipts remain historical.

The BDD tooling suite passed 268 tests. Stale capture-only and capture-host cases keep their exclusion or block. The
dependency map records a missing
Sketch3D runner explicitly and refuses a runtime binding for it.

One filtered Line scenario passed eight steps under Xvfb and Lavapipe against
the restored pin. Its shared background creates the sketch. The updated test
recognizes its projected origin and does not create a second part. The image
was inspected. Pointer input was injected into the frame loop, not delivered by
the OS. The two initial failures exposed test assumptions and are retained.
This diagnostic is not a verified acceptance receipt (#3978).
