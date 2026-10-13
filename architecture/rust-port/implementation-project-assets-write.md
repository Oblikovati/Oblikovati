# Persistent project asset edits — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0019/0020. This extends project-library loading and explicit reload.
The source behavior is `model/material/store.go` (`Store.Save`) and
`app/materials.go` in the Go application. The Rust owner reports write errors
and adopts the new library only after a successful atomic replacement.

The session can store an owned physical or visual project definition. A model
operation first prepares a candidate snapshot and checks project ownership.
Storage then writes the selected family through the existing atomic writer.
Only a successful replacement permits the session to adopt the candidate.
The other library file, project selection and document history are unchanged.

Before writing, storage reads and admits the current target file. If its typed
content differs from the registered library, the edit fails with a reload
requirement. Opaque record fields are retained through the shared document asset
merge, and legacy appearance fields are migrated through the existing converter.
The candidate must round-trip to the exact intended typed definitions before
the writer is called. No asset records or wire definitions are duplicated.

A single edit cannot change both families. This avoids a two-file partial commit.
Missing project selection or design-data location, wrong source ownership,
malformed files, changed libraries and failed atomic writes remain named errors.
The design-data directory must exist; directory creation and project setup
remain separate tasks. This does not add a new filesystem fallback.

Tests use the named memory filesystem. They check physical and visual write/reload,
legacy migration, opaque field retention, the untouched library's bytes, failed
write preserving memory and disk, retry, external-change refusal, missing project
configuration, source refusal, and independent model candidates.
They also check malformed and wrong-family file refusal, creation of a missing
library after reload, and retention of externally added opaque record fields.
Run `cargo test -p oblikovati-automation --test queries project_assets` and
`cargo test -p oblikovati-model --test project_assets`.

Canonical create/update API commands, document assignment, native editors and
project UI controls remain open. This supplies their persistent owner boundary;
it does not advertise incomplete wire methods. Kernel/exchange work remains
with the kernel. Validation runs on this Linux host only, without push or PR.
