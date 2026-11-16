# Fillet history probe

Base `9e7f0fddc`, pinned Martensite
`bef25f47548431c8f128f08f66a1f719bcee4982`, Rust 1.99.0. The initial
results below are headless kernel compatibility probes. Later sections record
desktop diagnostics; none establishes full BDD acceptance.

A 10-unit box with one 2-unit edge fillet produced a valid body and the
closed-form analytic volume. The test also checked seven result faces: six
planar areas, one curved area and one loop on each face. The curved face may
be represented as a cylinder or rational spline; the area check alone does
not certify its exact surface. Generated, modified and retained face history
covered every result face, but this does not prove unique durable names.

The box had 12 input edges and 15 output edges; direct retained or modified
history attributed 11 output edges. The remaining edges need an identity path,
for example from named neighbors, with ambiguity refused. A second probe copied
the host's prism and its selected edge, then checked a valid result, source
volume and face coverage. It did not resolve a document-scoped edge key.
At that stage the host feature, full provenance, replay, UI and scenarios
remained open (#4045); later sections record the work completed since then.

An additional cube probe checks exact face adjacency on every result edge.
Each result edge has two neighbors, and their source-history pairs are unique
for this case. The selected input edge is absent from the result and has no
modified edge in the kernel history. Its name therefore needs a deleted-entity
record; it cannot be rebound to an arbitrary result edge. These observations
do not establish unique neighbor-derived names for general topology.

The draft adapter now accepts exact named edges from a certified prism, copies
the input, builds and validates a detached result, and retains generated,
modified and deleted records for named input faces, edges and vertices. Four
focused tests cover one and two selected edges, analytic volumes, invalid
inputs, source isolation and stale-history rejection.

The draft adds reference scheme 3 for names derived from complete parent keys.
Parent order is canonical; duplicate, foreign, malformed, oversized and deeply
nested names are refused. Face, body and fillet inputs validate this scheme
with the same document scope as existing semantic names. The candidate now
assigns and resolves names for all result topology in the one- and two-edge
cube tests. Ambiguous direct history is refused. Names stay equal across two
radius values and after saving and reopening the source prism. The fillet
feature can now be replayed with the draft certified solid evaluator. A
headless test covers parameter-driven source edits, undo/redo, save/reopen and
resolution of an existing named face. It also verifies two sequential fillets
retain deleted edge records and prior body names.

The generic scene conversion now accepts fillet body meshes. A headless
viewport test creates a named fillet through ordered replay and converts its
result into a nonempty scene; a mesh test compares signed triangle volume to
analytic volume. Certified sketch continuity and datum preview use the solid
evaluator, and a continuity test resolves a datum after a fillet. The main
desktop scene and Part update owners now use the named-solid evaluator. A
desktop-owner test opens a saved fillet, builds a scene and selects both a
retained planar face and the generated curved face by exact key;
it checks that the curved-face key survives a width edit, undo and redo. The
UI controls were still under test at this stage. No frame-time result is
claimed for this work.

The common named-solid queries now expose exact face picking, topology counts,
face area and type, loop count, edge length and analytic bounds. The replay
test checks every cube fillet face against independent analytic areas, surface
class and loop count before and after edit and reopen. The desktop owner uses
these queries and retains the original body key across the replacement.

The session can now create a fillet as one reversible edit. A prepared request
parses a constant or named parameter and is shared between preview and commit;
the certified validator replays a private candidate before publication. Tests
cover valid and invalid real-kernel candidates, preview isolation, undo and
transaction abort. An analytic edge-to-view-segment query returns exact named
edge candidates for both prism and fillet bodies. Pointer selection now uses
that query. An owner test covers selection, toggle, clear and the selected-edge
outline.

The desktop work boundary now accepts selected edges for a private fillet
preview and a certified commit. An owner test covers preview, cancel, commit,
undo and redo without changing the document during preview. A dockable Fillet
panel now accepts the selected keys; GUI input tests cover opening its toolbar,
preview, Enter and Escape. A native Xvfb test clicks a model edge and its
toolbar and its configured marking menu, then previews, cancels, commits,
undoes, redoes, saves and reopens the part. `native-preview.png` records the
preview after moving the panel away from the model; the test checks that the
preview is visible at a projected face point. The test injects native-window
events and is not a complete scenario acceptance result. Preview computation
currently runs synchronously and its effect on frame time has not been
measured.

The Part context menu also shows Look At and Home View. Look At is disabled
for an edge alone and resolves a selected planar face or authored work plane.
A native-window test checks face orientation and Previous View; a separate
owner test checks work-plane selection. These tests do not cover every mouse
scenario or establish the frame-time target.

The native edge test now checks a green selected edge under the pointer, a
blue edge after pointer departure, and clearing selection with an empty click
that dismisses the Part menu. Display segments are grouped and cached per
selected edge; their screen distance controls color only. Exact selection
continues to use the analytic edge query.

The existing typed `features.add` Fillet request now resolves its exact edge
keys to one current certified body and validates the candidate before commit.
Unsupported forms are declined. The desktop commit uses that same typed path.
An API test covers invalid options and references, undo and redo; a CLI script
test creates a Fillet, recomputes, saves, reopens and recomputes it. Direct
Fillet feature editing through `features.edit` was open at that stage.

The typed `features.edit` path now exposes Radius as scalar zero and validates
the edited candidate with certified replay before it changes history. A real
desktop-owner API test checks a changed scene, invalid units, zero and oversized
radii, invalid scalar batches, undo/redo, a named parameter, and save/reopen.
The native Extrude distance-edit test and automation suite also pass. This is
API and owner evidence; the desktop tool was still open at that stage. The
later desktop diagnostic is in `../2026-11-17-fillet-ui-edit-4045/`.

An exploratory debug-build CPU timing of the 12-edge cube query used three
warmups and 30 samples: median 2.101 ms, p95 2.222 ms, maximum 2.223 ms. The
command was `cargo test -p oblikovati-martensite --test sketch_prism
measure_named_edge_pick_latency -- --ignored --nocapture`, with
`CARGO_PROFILE_TEST_DEBUG=0` and incremental compilation disabled. It does not
measure input delivery, display, a complete frame, a larger model or a release
build, and is not evidence of 60 fps.
