# Sketch inference preference API — partial R3

This step implements
[issue #3773](https://github.com/Oblikovati/Oblikovati/issues/3773).
The canonical `sketch.getInferenceOptions` and `sketch.setInferenceOptions`
methods work through file and session hosts. Read-only hosts expose the getter
and decline the setter. The API crate owns the DTO, client methods, method names,
and frozen priority enum IDs. No public contract is re-declared here.

The active document owns its preferences in the complete
[sketch settings record](implementation-sketch-settings.md). With no active
document, the workspace owns one fallback shared by all clients. The fallback is
session state and does not persist. It does not seed new documents. Closing the
last document exposes the previous fallback again. Defaults enable both toggles
and use `horizontalVertical` priority.

A set replaces both booleans. An empty priority retains the current enum value;
`parallelPerpendicular`, `horizontalVertical`, and `none` select the canonical
priorities. Other spellings decline before adoption, with the offending value
and expected choices. Stored unknown numeric enum IDs retain Go's `enum(?)` view;
an explicit supported priority can replace them. Get ignores unused request
arguments, as the Go handler does.

The Rust setter changes only the three inference fields. This corrects the Go
baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a` defect in
`app/document_sketch_settings.go::sketchSettingsFrom`: that function creates a
three-field record and resets grid, constraint-display, creation, and relax
settings to zero. Regression tests prove that the Rust setter retains all other
fields rather than copying that destructive behavior.

Document edits use normal history, including no-op detection, grouped commit,
abort, undo/redo, and observer veto. They make an absent record explicit, so a
first write of default inference values is still a document-state change. Saved
settings survive reopen and copy; later copy edits do not change the source.
Preference edits use explicit update mode and retain current sketch verification.
They do not run the solver or apply inference to existing geometry.

Both document and fallback paths prepare the response before adoption. A response
failure keeps the prior settings. Even a projection that ignores a validation
decline cannot adopt invalid preferences. File identity and allocator reservations
are not part of this metadata edit.

For example, use the canonical Rust client:

```rust
use oblikovati_api::wire::InferenceOptionsView;
let previous = client.sketch().inference_options()?;
let updated = client.sketch().set_inference_options(&InferenceOptionsView {
    infer_enabled: previous.infer_enabled,
    constrain_enabled: false,
    priority: "parallelPerpendicular".into(),
})?;
```

Tests cover priority parsing and projection, boolean replacement, malformed
requests, metadata retention, detached defaults, all document kinds, session and
file hosts, read-only refusal, shared fallback lifetime, history, response
failures, ignored declines, solver-state retention, veto, save/reopen, and copy.
The matching Go preference and router tests provide behavioral reference checks.

Line inference, line API creation, automatic relation reporting, interactive
snapping, and profile construction remain required work. This preference API
does not close R3 or a Martensite capability check.
