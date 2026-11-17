# Preference save race evidence (#4008)

The draft detects a change between loading a cached preference file and its
pre-write check. It does not make replacement conditional on that check.
`race-diagnostic.rs` injects a competing write during staging and immediately
before commit. Both tests fail: the save succeeds and can overwrite that write.

To reproduce, copy the diagnostic into `crates/persistence/tests/` and run
`cargo test --offline --locked -p oblikovati-persistence --test race_diagnostic`,
using the filename `race_diagnostic.rs`. Remove the copied test afterward.
The output records the run under the original filename.

An atomic rename does not provide a comparison with the previously read file.
An advisory lock protects only writers that participate in the same protocol.
A safe contract must cover every writer or use a supported conditional replacement
primitive; the draft does neither. Application and orientation preference saves
also remain exposed to a change after their read. No cross-process safety or
platform-wide guarantee follows from this draft.
