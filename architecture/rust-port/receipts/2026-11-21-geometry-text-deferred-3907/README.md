# Geometry Text deferred from PR #4184

All 21 scenarios in each source version have unbound steps.
The [pinned diagnostic](pinned-cases.json) and
[published diagnostic](published-cases.json) list the exact feature hashes,
the source line of each case and its first unbound step. The diagnostics used
the application tree at `297d374f3`. The published replay also had the local
ribbon guard present. It did not add a scenario binding.

The native Linux X11 [ribbon guard](ribbon-guard.log.gz) opened a Part sketch
and confirmed that Geometry Text is unavailable and point Text is enabled.
The [point Text test](point-text.log.gz) also passed placement, cancellation
and undo and redo through native input.
The current Rust model stores Text by sketch-point identity. The public API
at v0.165.0 describes Text at a point, and the current Martensite integration
builds glyph outlines for that placement.

The Go legacy tests at `aa8297d1b0ac33c024f19e54821536d76e8a447a` passed for
the point Text tool and its glyph profiles. The pinned and published scenario
sources define the line and arc behavior for the follow-up in
[#3907](https://github.com/Oblikovati/Oblikovati/issues/3907).

The behavior ledger excludes these 21 pinned rows from this PR and keeps
their exact source identities and expected results. The release inventory
records the deferral for both feature hashes. The diagnostics count as
scope evidence. There are zero passing Geometry Text scenarios.
