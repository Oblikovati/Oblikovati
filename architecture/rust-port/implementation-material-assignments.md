# Material and appearance assignment resolver — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
[ADR-0019](../decisions/ADR-0019-materials-and-appearances.md) supersedes ADR-0010
for the full asset boundary. The historical accepted record is unchanged.

The fixed Go `model/material/assignment.go` and `assignment_test.go` retain the
Go issue #1103 regression: an explicit part appearance wins over material appearance,
including an overridden body material. The Rust model now has one resolver with
that order: face appearance, body appearance, explicit part appearance, effective
material appearance, neutral default. It selects declared intent before lookup;
it does not try a lower-priority asset when the selected identity is missing.

Physical material resolution is separate: body material if assigned, otherwise
part material, otherwise unassigned. Changing visual intent cannot change density
or physical material. A material with no appearance choice uses the neutral
appearance. Missing material, selected appearance, or neutral identity is a named
failure. A valid visual override does not repair a missing physical assignment;
the physical query still reports its failure.

Assignments belong to one document. Body and face overrides use an opaque
persistent key plus the occurrence identity path. Equal key text in separate
occurrences cannot share an override. A face query from another occurrence is
refused. The reference constructor rejects empty/NUL components; it does not
invent geometric identity or prove body/face incidence. Host topology validation,
lost-reference diagnostics, and history remapping remain required before storing
or rendering live assignments. No proximity matching or geometric fallback is
added.

Empty asset IDs clear intent, as in Go. Invalid replacements leave assignments
unchanged. The resolver borrows the existing document/project/builtin asset
scopes and returns the model-owned typed asset. Scope resolution and assignment
precedence each have one owner; the renderer does not repeat these rules.

The fixed Go assignment tests pass on this host. Rust tests use the independent
Go asset-record fixture and cover the complete precedence chain, #1103 clearing,
material/appearance separation, occurrence isolation, source-scope priority,
missing selected assets, missing physics, absent neutral default, clearing each
entity override, invalid references, and atomic replacement failure. Run
`cargo test -p oblikovati-model --test assignments`, then `make gate`.

[Atomic assignment edits](implementation-assignment-edits.md) now cover document
history and portable package attachment. Assignment API commands, body/face
reference verification, occurrence workflow integration, renderer consumption,
health propagation, asset editors and analytic mass properties remain open.
This is the shared model resolver; it is not a claim of rendered-assignment or
complete material-feature acceptance. No kernel or exchange code is added,
Windows/macOS remain deferred, and no push or PR is included.

## Local validation receipt

On 2026-10-13, the final full `make gate` passed for all 13 workspace crates on
local Linux with llvmpipe and the local Vulkan validation layer. Model production
coverage was 98.83%; every crate exceeded 80%. Repository duplication was
0.62%. All eleven assignment tests passed. Formatting, clippy,
build, tests, doctests, documentation, SPDX, workflow and architecture checks
passed. The initial gate's Markdown line-wrap error was corrected; no code test
was weakened. The AMD native presentation fault remains open.
