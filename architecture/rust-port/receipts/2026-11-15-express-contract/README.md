# Express assembly contract and journey checks

The API contract at `a8e651bc537e516953c7d9313792719e5b474a25` defines the
Express open option, the reported mode, and Load Full. Its full `make gate`
passed on 2026-11-15. Go statement coverage was 84.5%; Rust line coverage was
97.73%. Go builds and Rust library checks passed for Windows amd64 and macOS
arm64 as cross-compiles, not native runs.

The jscpd 4.0.5 scan of Go and Rust sources and tests reported 1.97% duplicated
lines. The 5.4.1 scanner reported 5.34% for the candidate and 5.35% for its
parent, using the same five-line and 50-token thresholds. The different results
are retained rather than treated as equivalent; API duplication tracking is in
[Oblikovati.API #301](https://github.com/Oblikovati/Oblikovati.API/issues/301).

The native pinned Express journey ran on the host branch using the published API
commit above, after the display-quality, producer identity, and ribbon merges. It passed 13 of 14 steps: open a real cached assembly, display
its geometry, and load its components. The final step failed because Finish and
Simplify were absent from the full assembly ribbon; see
[issue #3963](https://github.com/Oblikovati/Oblikovati/issues/3963). This is a
diagnostic result, not scenario acceptance. The final capture was inspected.
The harness injects events into the frame loop and does not establish OS-level
keyboard or mouse delivery. No MCPBridge live test was made for this contract.

The saved display mesh is checked against the current desktop chord and angular
limits before registration and when an existing Express document is reused.
Focused persistence, session, API-host, and cached-to-full scene tests passed.
The desktop test also confirms a missing saved view reports an error and leaves
the existing document active. The scenario source was clean at
`48987af7b5cbb06515bc6f30385f508382f34064` and its document-opening feature
matched the current shared source by SHA-256. Producer identity is checked on new open and reuse;
[issue #3964](https://github.com/Oblikovati/Oblikovati/issues/3964) remains
open until the work is integrated and published. Express cache creation is
currently unavailable on macOS and Windows because their running executable
identity is not yet verified.

After integration with the ribbon branch, the GUI library suite passed 492
serial tests and strict Clippy passed for the GUI and head targets. The suite
failed when run in parallel because GUI contexts are shared; this existing test
isolation issue is tracked in [#3878](https://github.com/Oblikovati/Oblikovati/issues/3878).
The session lifecycle suite passed 453 tests and automation queries passed 755.
The documentation gate passed after splitting the combined ribbon file.
