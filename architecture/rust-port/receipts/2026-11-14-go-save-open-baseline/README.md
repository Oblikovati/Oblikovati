# Go save and open baseline

Reference: `../Oblikovati` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
`app/session_persistence_test.go` uses the real package store for its save and
reopen test. It checks the reopened part type and name, and a separate test
checks that an unsaved document needs a path. `app/file_ui_hooks_test.go` checks
MRU events and veto behavior with a fake store.

On Linux, `go test ./app -run '^(TestSaveActiveDocumentAsThenReopen|TestSaveActiveDocumentNeedsPathBeforeFirstSaveAs|TestOpenDocumentFromMRUHonorsVetoAndRecords)$' -count=1` passed. These tests do not establish native dialog behavior, duplicate-tab activation, a saved sketch round trip, or close-prompt interaction. The current Rust native receipts check the recorded scenario steps; they do not establish file compatibility or visual parity with Go.
