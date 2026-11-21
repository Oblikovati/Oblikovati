# Point scenario source sync

The live Point feature adds `@capture_only` to point-10. Its recorded sketch
fixture is still unavailable. The scenario is excluded from automated runs;
this change does not establish that its behavior passes. The other Point
requirements are unchanged.

The reviewed local snapshot is `e76cb749ed43c73a91c24854057da7c890041f51`.
It is a snapshot of an unversioned source, not an upstream commit. The old
receipts remain historical because their source revisions differ. The native
build-provenance gap in #3978 remains open. Tracking: #3887 and #3865.
