<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Current document-saving diagnostic

On 2026-11-15, the full document-saving feature ran at application commit
`2bb7cfa04` and scenario snapshot `05317bbb2d0809a532b29b7ff6f726a5722b865f`.
The snapshot was verified before and after the run. All 23 scenarios and 218
steps passed.

The run covers save and reopen, cancellation, overwrite choices, preview
controls and file-dialog navigation. The final captures and report are
retained here. The reopened-sketch capture and solid-preview capture were
visually reviewed; intermediate dialog states are checked by assertions.

The run used Xvfb at 1600 × 1200 with Mesa Lavapipe. The runner injects input
into the frame loop. This is diagnostic, not acceptance evidence: native build
provenance remains open (#3978), and OS-level input was not tested in this run.
