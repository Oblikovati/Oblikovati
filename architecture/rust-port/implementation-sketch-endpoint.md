# Owned sketch endpoint replacement — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/edit_trim.go`, `ExtendLine` and `reshapeTrimmed`,
at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The model can replace either line end with a new hidden defining point at a
supplied finite seed position. The operation retains the line ID, former point,
other endpoint, entity order, construction roles, centerline role, and authored
constraints. Other curves that share the former point retain their references.
The report names the line, former point, and new point. It does not report an
intersection, a solved position, or a completed extension.

Validation rejects missing lines, non-finite positions, and positions equal to
either original endpoint before reserving an ID. The shared reservation boundary
rejects zero, reused, and exhausted IDs. Adoption uses a complete candidate;
failed adoption cannot publish an orphan defining point. Coordinates are stored
without projection, normalization, or arithmetic. Signed zeros and extreme
finite values are preserved. This is an ownership operation, not a replacement
for kernel geometry validation or intersection selection.

Line split now uses the same endpoint rewiring function. The duplicate split-only
setter is removed. The document operation uses the existing permanent reservation
and geometry replacement boundaries. No API contract, wire operation, recipe
schema, dependency pin, or UI command changes.

Session edits solve pending sketches after the edit callback. Thus constraints
can move an endpoint away from its supplied seed position. The session test
replaces a horizontal line's end with an off-line seed and requires the retained
horizontal constraint to restore its Y coordinate. The fixed former endpoint
must satisfy the same solve policy. Exact coordinate preservation is asserted
at the model boundary, before solving; undo and redo restore exact evaluated
states. A future extension command must validate its final solved result before
claiming geometric incidence. The identity-only report makes no such claim.

Tests cover both ends, shared ownership, preserved roles and constraints, hidden
point IDs, recipe reload, exact stored coordinate bits, invalid input before
reservation, rejected allocated IDs, typed allocator exhaustion, missing document
owners, solve behavior, undo/redo, veto, group abort, and failed reply preparation.
Rejected or undone edits retain consumed IDs. The existing native line-split test
also exercises the shared rewiring path through the typed API and presented image.

Kernel `bb1e00115` supplies exact endpoint and projection ordering, but is not in
the app's v0.2.5 release. Certified intersection selection, release intake, trim
and extension commands, and their native acceptance remain open. No local kernel
source override or app geometry engine is added.

The full local `make gate` passed across all 13 application crates, with all
targets and features, formatting, Clippy, build, tests, documentation tests,
coverage, Markdown/link checks, SPDX checks, workflow lint, and duplication.
Native tests ran serially on an isolated 1920×1080×24 Xvfb display with software
Vulkan and validation layer 1.4.321. Every crate exceeded 80% line coverage.
Model coverage was 98.78% (17,998/18,220); endpoint replacement was 92%
(69/75), and its document boundary was 95.65% (22/23). Duplication was 0.75%
(510/68,157 scanned Rust lines). The Go baseline extension and front-trim tests
also passed. Those Go tests check the reference operations; they are not proof
that the Rust extension command is implemented.

Cross-platform and comment review found no new numeric tolerance, platform-sized
identity, unordered allocation, geometry perturbation, or removed intent comment.
The model tests require exact stored bits at extreme finite magnitudes; the session
test uses the existing solver policy for achieved residuals. Windows and macOS
runtime acceptance and full R3 completion remain open. No push or PR is included.
