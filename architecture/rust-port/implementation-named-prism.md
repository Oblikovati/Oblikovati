# Verified prism reference names — partial R2/R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
[ADR-0021](../decisions/ADR-0021-semantic-topology-reference-keys.md) defines the
scheme-2 semantic name format. Existing Go scheme-0/1 bytes retain their meaning.

`retain_named_sketch_prism` evaluates an authored feature from a sealed document
history snapshot. It uses the existing prism construction history to name the
body, start cap, end cap, and each side face. Source sketch and line recipe IDs,
feature ID, and document GUID define the namespace. Coordinates, traversal order,
display names, and native addresses do not contribute to key bytes.

Each face name is written to the exact binding index, read back, and checked
against the retained native face. Duplicate roles are refused. Face queries
integrate the bound analytic B-rep face. The adapter retains native handles
privately and requires the current owning history for every query and enumeration.
An edit, undo, close, or different history owner makes an old binding unusable.
A rebuilt result can bind the same saved name to its current face.

The desktop now uses `NamedPrismEvaluator` inside its ordinary part replay. The
model passes the authored feature ID to the backend. Each result is named and
verified before the completed receipt can enter `PartEvaluationSlot`. Rendering
and exact queries borrow that same retained result; naming does not trigger a
second kernel evaluation. The mesh-source interface exposes only display output,
not a raw native body or an unguarded exact-query route. The unnamed diagnostic
backend remains a test/probe adapter and is no longer the desktop backend.

Host tests check separate namespaces for two features with identical geometry,
stale-result refusal before refresh, stable names after a parameter edit and undo/redo, and
analytic face queries against the displayed body. A failed feature removes its
current names while independent features remain available. Scope-only capture
retains the same generation and close checks without copying the recipe.

Tests check all six rectangle faces independently, changed depth and profile
width, undo invalidation, wrong namespaces and entity kinds, unknown sources,
and saved package recipe reconstruction under a new history owner. A holed
negative oblique sweep retains names after rotation of boundary traversal.
Scheme tests assert exact bytes, count limits, malformed/truncated payloads,
UUID ownership, and non-aliasing with legacy keys. The existing OCCT prism
oracle and degenerate-input tests remain part of the full gate.

This adapter covers the existing selected-line prism capability. General edge
and vertex naming, split/merge and healing provenance, occurrence namespaces,
Go topology migration, picking, and assignment API wiring
remain open. No new public wire methods are advertised. No geometry algorithm
or tolerance changes are included. Net changes in geometric recognizers,
tolerance constants, fallback sites, and native type assertions are all zero.

Run `cargo test -p oblikovati-martensite --test sketch_prism`,
`cargo test -p oblikovati-model --test semantic_references --test reference_bindings`,
and `make gate`.

Local validation uses an isolated Xvfb display, serial tests
(`RUST_TEST_THREADS=1`), Mesa software Vulkan
(`VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json`), and the installed Vulkan
validation layer at `.tools/vulkan-validation-1.4.321/layers`. The desktop-display
attempt stopped in the native head smoke test because its window did not map
within three seconds. The same three native head tests pass on Xvfb without
changing their assertions or timeouts. This does not certify desktop hardware
presentation; see the existing [presentation diagnostic](diagnostic-native-present-pixels.md).

A concurrent Xvfb gate attempt terminated the head library test process with
SIGSEGV. Three shorter concurrent runs of its `native::` tests failed pixel
comparisons; a separate run also reported a display connection error. All 17
native tests passed when run serially on the same server. This supports a
shared-display test conflict but does not establish the segmentation fault's
cause. Concurrent native acceptance remains open; no assertions, timeouts, or
application behavior were changed to obtain the serial result.
