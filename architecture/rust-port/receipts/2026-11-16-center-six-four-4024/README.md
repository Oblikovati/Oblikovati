<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Centered rectangle six-by-four diagnostic

The focused native frame-loop diagnostic for `rectangle-center-13` passed after
binding the current scenario wording. It checked the displayed annotations,
four sides, two construction diagonals and common diagonal midpoint. It also
checked the model-axis extents: 6 mm along the first sketch axis and 4 mm
along the second. The retained report is `gap-report.json`.

The scenario snapshot was `e76cb749ed43c73a91c24854057da7c890041f51`,
verified before and after the run. The test uses a new YZ sketch and a screen
position in place of the unavailable recorded fixture and center. It does not
prove the recording's fixture or an OS-level input path. This is diagnostic
evidence, not a verified acceptance receipt (#3978). The full gate was not run.
