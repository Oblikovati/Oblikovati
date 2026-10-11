# CLI open and Save As

Tracking: [#3753](https://github.com/Oblikovati/Oblikovati/issues/3753).
Source: Go `cmd/oblikovati-cli/cmd_inspect.go` and `cmd_saveas.go`, revision
`aa8297d1`. Storage and session ownership remain ADR-0003 and ADR-0004.

`open <path>` reconstructs the supported parameter, datum, sketch, property,
attachment, resource, interest, and attribute recipes through FileWorkspace.
The command reports the current kind and display name only after registration.
It does not write the source. Unknown document kinds use the same named decline
as new-document creation. Manifest-only `info` remains a separate inspection.

`save-as <src> <dst>` opens through this same owner, then uses its atomic Save As.
The source kind determines the API-owned extension. Matching extension case and
leading-dot filenames follow the shared Go-compatible extension function used
by `new`. Both commands retain native Linux path bytes. Save As preserves the
document GUID and advances its save revision. An unchanged recipe retains its
model revision. Derived titles follow the new path; authored titles remain.
Source bytes remain unchanged unless source and destination are the same path.

Invalid arguments, reconstruction errors, identity errors, and storage errors
produce failure without success output. Atomic storage failure retains any old
destination. Output errors propagate after storage and do not undo a committed
file. No second package-copy or identity pipeline is introduced.

Named fake file and GUID sources cover all four kinds, malformed recipes,
unknown kinds, storage phases, identity failures, output failures, retained
streams and model sections, and reconstructed parameters. Native Linux tests
exercise new/open/save-as/reopen, source preservation, same-path replacement,
save and model revisions, invalid UTF-8 paths, malformed legacy features, and
missing destination directories. Go `TestSaveAsCopiesPackage` and
`TestOpenReportsIdentity` pass at the fixed source revision.

Full Linux `make gate` is the acceptance command. Kernel geometry replay and
native display remain open. Nonempty legacy solid feature programs are refused
before saving; these commands do not claim to reconstruct their B-rep geometry.
