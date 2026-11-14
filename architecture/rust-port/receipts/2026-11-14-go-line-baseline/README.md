# Go line baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

On Linux, `go test ./app -run '^(TestLineToolAndCancel|TestFinishedLineChainCommitsAndReleases|TestCancelledLineChainReleasesWithoutRecording)$' -count=1 -v` passed. These session tests cover line tool use and line-chain transaction completion and cancellation. They do not establish native-window rendering, the recorded numeric measurement, or every line scenario. The Rust native receipts cover the selected pinned steps and inspected final captures; they do not establish visual parity with Go.
