# Document-context chain preview

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This consumes the [inferred chain model operation](implementation-inferred-line-chains.md).
Kernel/API pins remain v0.2.5/v0.158.0; there is no local source override.

The session and desktop owner now expose inferred-chain preview and commit.
Both use the same chain authoring operation and the existing recipe-edit path,
including exposed parameter refresh and pending sketch recompute. Preview reads
current document parameters, projection sources and inference settings, including
changes inside an open edit group. It does not use an empty scratch sketch to
infer against existing entities.

History preview uses a private current-state copy. It restores all reservation
namespaces, including deleted parameter, sketch, entity, constraint, dimension
name, feature and derived-table marks. The private copy uses normal edit
validation. The original owner retains its geometry, generation, open token,
Undo/Redo lists and reservation marks on success or failure. Session preview
emits no transaction or document events. It therefore predicts the same IDs and
solver ordering as the next commit when the source is unchanged and no
intervening edit consumes IDs.

Preview and commit require a current converged result before publication.
The readiness check shares the existing feature check's parameter freshness,
projection freshness and numerical convergence logic. Authoring permits free
variables; feature use still requires zero DOF. A conflict introduced by inferred
constraints is a typed readiness error. Commit retires any consumed IDs while
rejecting the candidate; failed preview consumes only its private reservations.

The viewport packet now selects one explicit source: a standalone recipe or a
resolved private-document result. The latter carries the solved geometry and
only the authored member IDs. The renderer does not solve it again. Both sources
use the same member sampler and native structure builder. The sampler obtains
construction roles from the actual owned geometry and emits transient member
indices; source IDs do not become persistent selection keys. Existing entities
are available to inference but are not repeated in the preview overlay.

Tests compare preview and commit geometry and inference records exactly. They
cover shared points, projection of the origin, changed parameter values,
automatic constraint settings, open transactions, Redo, all reservation
namespaces, repeated previews, invalid requests and contradictory inference.
The existing standalone recipe, numerical-policy and native cache checks remain.
Resolved previews also exercise Fit exclusion and resampling after zoom.

Native tests extend an existing fixed rectangle with open and closed chains.
They verify unchanged source geometry and generation during preview, exact
solved geometry on commit, visible member keys, Undo/Redo and save/close/reopen.
Malformed replacement retains the last valid overlay. Captures are checked
against mapped X11 pixels. The inspected images are
`/tmp/document-chain-preview-true.png` and
`/tmp/document-chain-committed-true.png`. Both retain the same snapped chain.
The green diagnostic prism remains during preview. After commit, its existing
whole-sketch readiness check reports `FreeVariables(2)` and removes the solid;
the native test asserts that named failure and confirms that Undo restores it.
This is not general profile or extrusion acceptance.

Continuous mouse chaining, final-segment field locks, the tool-lifetime
transaction, Enter/Escape completion, Close and local Undo remain open. This
step supplies their shared preview/commit operation; it does not expose a
reduced interactive Line tool or complete R3/R4. General named regions and the
other kernel blockers retain their existing priorities.

Cross-platform and coherence review: preview preserves namespace ordering,
uses document working units and the injected solve policy, and adds no numeric
tolerance or geometry algorithm. Native sampling uses the existing deflection
policy and point cap. The Go inference and ZFit exclusion comments remain.
Temporary screenshot code was removed before the full gate.

## Local acceptance

The complete `make gate` passed for all 13 application crates, all targets and
features, including native tests on Xvfb with software Vulkan. It includes
formatting, Clippy, build, tests, doctests, coverage, Markdown, documentation
links, SPDX, workflow checks and duplication. The retained local log is
`/tmp/document-chain-preview-gate-cleanup.log`.

Every crate exceeded 80% line coverage. Model reached 98.85%, session 98.68%,
the kernel adapter 91.67%, viewport 94.28% and head 96.19%. Duplication is
0.77% (577 of 74,756 scanned lines, 67 pairs). This is ten fewer duplicated
lines and one fewer pair than the preceding local integration. Acceptance
covers Linux; Windows and macOS remain deferred. No push or PR is part of
this change.
