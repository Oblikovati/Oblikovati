# Rust attachment implementation

## Session ownership of attachments — partial R2

Tracking: [#3691](https://github.com/Oblikovati/Oblikovati/issues/3691).
`DocumentSnapshot` now owns the attachment collection. Collection and visibility
changes participate in transaction/no-op detection, undo/redo, abort, refresh,
and save baselines. Open and template creation install validated attachments;
malformed records fail before GUID allocation or identity registration.

Save and save-copy capture live attachment records through the validated codec.
A failed write preserves live state and identity; copy leaves the source dirty.
Reopen restores embedded bytes and IDs, linked nanosecond timestamps, order, and
visibility without consulting the original external files. Attachment metadata
remains outside the existing model-only digest.

Tests cover visibility-only history, no-op updates, commit veto, refresh,
attachment-only dirty state, normal/copy save and reopen, failed save, template
independence, and malformed input with no identity claim. The attachment API and
project-path resolution remain pending.

## Attachment API — partial R2

Tracking: [#3692](https://github.com/Oblikovati/Oblikovati/issues/3692).
The typed host now implements `documents.listAttachments`, `addAttachment`, and
`removeAttachment`. All select an explicit document ID. File-host capture uses
`with_attachment_files` to inject foreign-file access; embedded capture requires
that provider. Linked/generic discovery without a provider reports unknown.
Each listed attachment uses one probe for both status and resolved filename.
The timestamp formatter is shared with persistence, preserving nanoseconds.

Add and remove use transaction ownership and encode their response before commit.
A read, status, or encoding failure discards the candidate. Commit veto also
restores model state and history, though source reads and GUID allocation may
already have occurred during candidate construction. Removal works through the
shared edit host; capture remains a file-owner command.

Four typed-client tests cover all kinds, source changes and absence, metadata
errors, embedded ownership, required providers, invalid requests, duplicate names,
commit veto, undo, and save/reopen. They also verify immutable-session discovery
and that these calls do not change focus. Go router tests provide reference
evidence. The shared inventory has 30 queries, 23 edits, and six transaction
methods; the file owner adds ten commands. Project-path resolution and native
transport remain pending.

## Project attachment lookup — partial R2

Tracking: [#3693](https://github.com/Oblikovati/Oblikovati/issues/3693).
`ProjectAttachmentFiles` wraps the foreign-file provider with the existing
project lookup. Relative references search the workspace and then libraries;
absolute references bypass roots. Empty references and unconfigured roots do
not use the process working directory. Access errors stop the search.

A successful observation carries both its resolved path and modification time.
Discovery consumes that observation without a second metadata read. The stored
reference remains unchanged. Embedded capture reads the selected path and owns
the resulting bytes after the source is removed. Non-UTF-8 candidate paths are
refused explicitly because the attachment contract uses UTF-8 strings.

Tests cover search order, missing and absolute paths, access failures, one probe
per candidate, payload selection, embedded ownership, and typed API results with
different stored and resolved names. This adapter is injected into `FileHost`;
application startup and project configuration wiring remain pending.
