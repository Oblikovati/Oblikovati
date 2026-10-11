# Per-document sketch authoring settings — partial R3

This step implements
[issue #3772](https://github.com/Oblikovati/Oblikovati/issues/3772).
The snapshot owns an optional canonical API `SketchSettings` value. It retains
inference toggles and priority, grid spacing and subdivisions, constraint display
and creation options, and relax options. Application-wide sketch preferences
remain a separate owner.

`DocumentSnapshot::effective_sketch_settings` returns a detached value. An absent
record uses `default_sketch_settings`, with the Go defaults from baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. Reading defaults does not create a
record. Explicit false and zero values remain explicit. A present partial legacy
record uses zero values for missing fields, as the actual Go converter does.
The comment in `yamlcodec/yamlcodec.go` claims a default overlay, but
`persistence/fromCodecSketchSettings` does not apply one. This port preserves the
observed conversion rather than that comment.

The package codec uses the API type and its frozen numeric enum IDs. It refuses
invalid field types and nonfinite grid spacing. Unknown numeric enum IDs remain
contract values; their operational interpretation is a separate API boundary.
Edits preserve unknown fields in the retained settings mapping. Unchanged values
retain the original section. Clearing the explicit record removes that section.
No other section is replaced, and settings do not enter the existing model-only
recipe digest, which follows the Go package boundary.

Open validates settings before allocating document GUIDs or registering identity.
Creation restores them before owner registration. Snapshot capture writes current
settings through normal save and copy. Dirty comparison, no-op detection, undo,
redo, group abort, and refresh include the optional settings value. An explicit
default record differs from absence, even though their effective settings agree.
A commit veto discards the candidate through the existing transaction boundary.

For example, a typed command can store document defaults:

```rust
use oblikovati_model::sketch_settings::default_sketch_settings;
commands.run("Sketch defaults", |snapshot, _| {
    snapshot.sketch_settings = Some(default_sketch_settings());
    Ok::<(), String>(())
})?;
```

Tests cover every default field, detached reads, explicit zero values, complete
and partial legacy records, extension retention, malformed load before identity
allocation, dirty state, no-op history, undo/redo, abort, veto, save/reopen, copy,
independent documents, part/assembly/drawing owners, and refresh comparison.

The [inference API step](implementation-sketch-inference-api.md) adds canonical
get/set dispatch and the shared no-document fallback. The inference engine, line
creation, interactive tools, and profile construction remain open. This step does
not close R3 or any kernel integration check.
