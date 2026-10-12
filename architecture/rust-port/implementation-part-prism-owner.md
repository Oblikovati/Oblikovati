# Stored prism inputs in the part owner — partial R1

The document now stores sketch prism inputs in the same ordered feature program
as work points, axes, and planes. `PartFeatureDefinition` owns both kinds. The
existing datum names remain aliases. There is one graph, serializer, feature-ID
allocator, edit history, and package program. The historical `work_geometry`
field and package section names remain for format compatibility.

`DocumentEditHistory::create_sketch_prism` reserves the permanent feature ID and
stores the stable sketch, directed line selections, and displacement sources.
Undo restores the inputs without reusing reserved IDs. Creation does not build
geometry. Candidate edits refresh source dependencies before adoption, including
host datum edges after sketch rehosting. Missing or forward dependencies refuse
the candidate. Initial owner adoption checks stored dependency coverage.

The common program writes version 16 when it contains prism inputs. The source
entry uses version 11 and embeds the existing version 1 prism recipe. Datum-only
program versions remain unchanged. A version 15 reader refuses the new entry.
Package tests preserve unknown sections, exact encoded bytes, and the permanent
feature high-water mark. Numeric solve evidence is rebuilt after reopening.

The diagnostic `retain_stored_sketch_prism` adapter reads active stored inputs
and calls the existing selected-profile kernel path. It refuses absent, cutoff,
manually suppressed, conditionally suppressed, failed-condition, and non-body
entries. Local tests change the depth through the owner, check analytic kernel
results, undo the change, and check that an earlier retained result is unchanged.
Datum API rows retain their own indices and exclude body inputs.

This is authored-input ownership and diagnostic kernel consumption. It is not
the complete body evaluation, feature health, cache, or extrude API contract.
The datum projection names a body-scope decline and keeps independent datum
frames available. General profiles, feature result provenance, persistent naming,
complete validity, and the remaining extent and operation modes remain open.

Run `CARGO_NET_OFFLINE=true make gate` for the complete local application checks.

## Native owner-to-scene acceptance

The native Linux prism test now reads the stored feature ID from the document
owner. It creates the input in a transaction and changes the driving width in a
second transaction. The current solved document feeds the released kernel and
its derived mesh feeds the viewport. Five plane frames use this same path.

Undo reproduces the original GPU pixel bytes; redo reproduces the edited pixel
bytes. Each submission advances the scene generation. Every completed capture
also matches the actual X11 drawable, including the blue pixels through the hole.
This extends the diagnostic acceptance path. The desktop's general model-to-scene
publication, feature result contract, and interactive feature commands remain open.
