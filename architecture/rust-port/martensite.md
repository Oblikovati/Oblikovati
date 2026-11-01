# Martensite integration and readiness

This plan consumes released crates from
[Martensite](https://github.com/Oblikovati/Martensite). It does not assume that
all OCCT capabilities are already usable. It also requires Martensite to gain
the Go-kernel capabilities needed by Oblikovati that OCCT does not provide. The
[source baseline](source-map.md) fixes the inspected revision. The active
[Martensite v0.5.0 migration](martensite-v0.5.0-migration.md) records the
adapter pin and the current native-presentation gap.

The accepted application contracts are
[ADR-0007: Martensite integration](../decisions/ADR-0007-martensite-integration.md)
and [ADR-0008: Vulkan renderer](../decisions/ADR-0008-vulkan-renderer.md).
Their acceptance does not close any capability check below. Required new
kernel-internal decisions still belong to Martensite.

## Observed integration limits

- Modeling is consumed from released toolkit crates, including
  `martensite-tkbrep`, `martensite-tktopalgo`, `martensite-tkprim`, and
  `martensite-tkmesh`. The adapter acceptance is separate from toolkit
  acceptance in Martensite.
- `Shape` contains a shared `Handle<dyn TShape>`. A cloned shape preserves
  shared identity. Some builders can update referenced geometry while preparing
  an operation, so `&Shape` alone does not imply immutable evaluation.
- The released v0.5.0 graph has no V3d or Vulkan presentation crate. The
  viewport remains on the prior renderer until issue #3630 moves it to a
  released Martensite-native presentation path. Do not extend that temporary
  renderer dependency.
- The kernel completeness record does not claim full behavioral equivalence
  or a complete platform matrix. Toolkit mapping counts are not readiness
  evidence for an application feature.
- The kernel's current priority is visualization and Vulkan before broad
  exchange-format work. The application schedule follows that order.

Select the required toolkit dependencies in the application adapter. Keep their
release commit identical. A later facade expansion belongs in Martensite; it is
not a prerequisite for every initial application test. Do not invent a
dependency named `martensite` that the repository does not provide.

The Go model and API use centimetres and radians by default. Documents can
also carry a working scale (`model/param/units_of_measure.go`). Preserve its
length, area, and volume exponents, including tolerance conversion. Do not
assume millimetres at the Martensite seam or apply a hidden factor of ten.
Test non-default working scales and exchange units before accepting K01.

## Required extensions from the Go kernel

User requirement, 2026-10-10: **port and adapt into Martensite every Go-kernel
capability that supports an application feature and is outside OCCT.** These
extensions are part of the application port's required scope.

R0 traces feature-to-kernel dependencies and assigns each required capability
an integration path:

| Capability state | Required action |
| --- | --- |
| Equivalent behavior is available in Martensite | Reuse it and verify the full application contract. Do not port a duplicate algorithm. |
| OCCT supplies the behavior, but Martensite does not yet supply it | Complete the Martensite OCCT port and test it against the OCCT oracle and application cases. |
| The Go kernel supplies required behavior that is outside OCCT | Port and adapt the Go implementation into Martensite as an extension. Retain its application consumers and acceptance requirements. |

Classify behavior, including options, failure modes, provenance, and numerical
requirements. Do not infer absence from a missing API name. Uncertain mappings
stay open until source and tests establish the correct path.

Each required extension must:

1. Record the Go source files and commit, consuming application features,
   existing tests and fixtures, and evidence that OCCT lacks the required
   behavior. Reuse the existing Go logic where valid; record corrections to
   known defects with regression tests.
2. Have a Martensite work item linked to the ADR for its kernel boundary, with
   its interface, dependencies, acceptance cases, and blocked application
   milestone. Consolidate decisions for the same boundary in one current
   record. Record the extension in Martensite's scope and design documents;
   link its application integration under the [Rust ADR policy](../decisions/README.md).
3. Adapt to Martensite's geometry and topology representation, ownership,
   tolerances, validation, diagnostics, and generated/modified/deleted history.
   Integrate with its general operation pipelines and shared algorithms.
4. Port the relevant Go tests and add independent correctness checks, generated
   degenerate inputs, and application-level tests. Differential agreement with
   the Go implementation alone cannot prove correctness.
5. Be integrated and tested in Martensite before the application feature
   can count as ported. Keep kernel algorithms out of the application adapter.

Track extension acceptance separately from OCCT parity. These capabilities
have no OCCT equivalent to count as mapped; do not inflate or reduce the OCCT
coverage denominator. Both tracks are required where an application feature
depends on them. The application inventory must link to their kernel evidence.

Use OCCT or SolveSpace for suboperations they can verify. For behavior neither
provides, define an independent oracle, analytic expected result, or proved
invariant with explicit hard assertions. Preserve the Go baseline as a
behavior reference. An unavailable OCCT comparison is not permission to omit
the extension or weaken its acceptance checks.

## Exchange ownership

[ADR-0018](../decisions/ADR-0018-exchange-format-ownership.md) assigns every
exchange format implementation to Martensite. DWG uses
[acadrust](https://github.com/hakanaktt/acadrust) in the kernel. This includes
legacy recipe/domain translators; they are not application port tasks.
The kernel owner provides the exchange contract, dependency pins, format tests,
and released package. The application consumes that release and tests its
commands, typed API, document transactions, resources, and diagnostics.
K10 remains open until both kernel evidence and application integration pass.

## Kernel capability checks

Every row starts as **unverified for Oblikovati**. Close a row only with an
executable application-level test, a pinned Martensite revision, the relevant
oracle evidence, and recorded results on its required platforms. Existing
kernel evidence can support a row, but cannot replace the application test.

| ID | Required capability | Candidate toolkit or module | Acceptance and first consumer |
| --- | --- | --- | --- |
| K01 | Units, transforms, tolerances, shape ownership | TKernel, TKMath, TKG2d/TKG3d, TKBRep | Establish API/model units and matrix conventions. Prove input isolation, copy behavior, orientation/location, and stable iteration. R1. |
| K02 | Curves, wires, profiles, faces, prism/revolve | TKGeomBase, TKTopAlgo, TKPrim | Build a closed parameterized profile, including an inner loop; extrude and revolve it. Check analytic faces, loops, orientation, and operation history. R1/R3. |
| K03 | Validation, diagnostics, analytic queries | TKTopAlgo, relevant geometry and Boolean modules | Check topology, geometric consistency, self-intersection, achieved tolerance, and analytic area/volume. An unavailable validation level is not a pass. R1/R3. |
| K04 | Generated, modified, and deleted history | Operation makers, TKBRep/TKBO history | Track each output entity to input parents and semantic feature roles. Resolve a named face after parameter edits, undo, and replay. R1/R3. |
| K05 | Meshing and exact pick references | TKMesh, TKV3d | Tessellate holes, seams, and curved faces deterministically. Map display picks to analytic entities without making modeling decisions from facets. R1V/R4. |
| K06 | Native viewport and UI composition | TKService, TKV3d, `martensite-vulkan` | Native presentation, resize, DPI, selection/highlights, lines/text, overlays, screenshots, and UI composition. R1V/R4. |
| K07 | Boolean and feature operations | TKBO, TKBool, TKFillet, TKOffset, TKHelix, related crates | Join/cut/intersect, sweep/loft, blend/draft, shell/thicken, pattern, surface and sheet-metal inputs. Check each operation family before its R5/R6 feature. |
| K08 | Assembly instances and HLR | TKBRep location, TKV3d, TKHLR | Shared part presentations with separate occurrence paths; exact HLR and drawing references after edits. R6. |
| K09 | Materials and full application display | TKService/TKV3d/Vulkan plus agreed extensions | Complete Oblikovati's display, appearance, point-cloud, lighting, and OpenPBR inventory. Separate app requirements from the kernel's legacy-renderer scope. R7. |
| K10 | Kernel-owned file exchange | acadrust for DWG; TKDE, TKXSBase, and kernel-owned format modules | Kernel tests/releases each reader and writer. Application verifies routing, units, assembly/metadata adoption, resources, and diagnostics against that release. ADR-0018; R8. |
| K11 | Required Go-kernel capabilities outside OCCT | Martensite extension modules identified by the R0 inventory | Port and adapt each required capability, preserve source/test provenance, and pass independent correctness and application tests before closing its dependent milestone. Applies across R1-R8 and the R9 cutover. |

Kernel gaps get a Martensite work item or porting-ledger entry with the source
case, operation, expected result, and dependent application milestone. Keep the
application feature pending until its required rows pass. Continue independent
model, storage, or UI work while the dependency is repaired.
Absence from OCCT cannot classify a required Go capability as excluded work.

## Operation result interface

The application-owned kernel module must provide these facts together:

1. A valid result shape or a typed failure that names the failed operation and
   entities. Include partial-result information where the contract needs it;
   never publish an invalid solid as a successful feature.
2. Generated, modified, and deleted entities with parent sets. Preserve existing
   identities when an operation does not change them.
3. Achieved tolerances and validation findings. Distinguish modeling tolerance,
   sketch solve precision, tessellation error, and display precision.
4. Analytic query results with their quality and diagnostics. Do not substitute
   facet-based mass properties, classification, or section curves.
5. A presentation source that can be regenerated. A failed presentation does
   not silently change a model result, and an old capture cannot report a new
   operation as successful.

Implement one evaluation path for each operation. Validation is its
postcondition, not an optional UI step. Keep topology predicates, repair,
meshing, and numeric tolerance decisions inside Martensite. The adapter maps
units, IDs, ownership, options, and findings; it does not repair a result with
another application-side geometry engine.

Martensite preserves OCCT behavior, while Oblikovati has additional exactness,
determinism, and provenance requirements. Native OCCT agreement alone does not
prove those stronger requirements. Audit them explicitly. If a required kernel
path violates them, record and fix the general path in Martensite, with the
decision in the owning kernel-boundary ADR and supporting oracle cases. Do not
loosen an application check to conceal the gap.

## Persistent naming

The application owns feature IDs, occurrence paths, document reference keys,
and their versioned encoding. Martensite owns the topology and operation
history that establish where each entity came from. Translate that history
once into the application's provenance records.

Do not make an OCAF document graph or TNaming store a second authority for
Oblikovati's feature history. Use a kernel facility only when the chosen
interface and tests need it. One document history and one persistent naming
mechanism must remain authoritative.

The initial naming test uses a face produced by an extrusion and then a
dependent sketch or feature. Change the extrusion length and upstream profile,
recompute, save, reopen, undo, and redo. The reference must still name the
intended entity or return a specific lost/ambiguous-reference error.

Add split, merge, deletion, symmetry, and multiple-body cases before broader
feature completion. Include the parent relation in both directions where
needed to name lower-dimensional entities. Bound naming recursion, detect
cycles, and define a total order for ties. Confirm each key by resolving it
when it is written. No geometric-nearest fallback counts as exact identity.

## Thread and resource checks

R1 must exercise two independent documents, shared immutable recipe inputs,
cancellation, and repeat evaluation. Check which operations mutate or cache
state reachable from input shapes. Choose explicit deep copies or isolated
evaluation ownership where required, and measure the cost.

Keep viewer objects separate from evaluation objects. Return thread-safe
immutable results across the seam only after their ownership is established.
If a necessary result cannot move safely, keep it on a dedicated owner and
send typed requests and values. This is an ownership choice, not permission
to serialize geometry into a lower-quality substitute.

For the viewport, define the owner of the Vulkan device, queues, swapchain,
offscreen images, UI resources, and fences. Establish one submission and
retirement protocol. Do not allow an ImGui backend to create a conflicting
device lifecycle or keep an image alive after its owning view is removed.

## Current sketch-region integration evidence

The [released region probe](implementation-profile-regions.md) checks eleven
bounded faces and their source-edge history against local OCCT 8.0.0. It also
reproduces the general-fuse rejection of a single closed circle or ellipse.
K02 remains open. A kernel-owned general sketch-region contract must cover
these valid inputs and the required exact/filtered predicates, tolerances, and
provenance. Its owning work item/ADR and release acceptance remain pending;
the application must not add a separate region algorithm to pass this gate.

## First kernel and viewport proofs

R1 produces a kernel-level fixture: a rectangular profile with a hole from
explicit dimension values, an extrusion, valid face/edge topology, analytic
properties, and provenance. The constraint-driven sketch is added in R3.
Vary size, orientation, and units. Include failed and near-degenerate inputs.
Compare the expected topology and analytic faces with the pinned OCCT oracle.

R1V starts from Martensite's checked offscreen examples, then proves a native
window and UI composition. After K05 is ready, replace the test presentation
with the R1 shape. Pick a face, obtain its persistent reference, resize the
window, edit the model, and verify selection and capture again.

A native window gap stays in the kernel/platform dependency list. Do not
create a second production renderer from the Go head to pass the milestone.
Only the narrow UI/window binding may remain native. Preserve the existing
application assets and UI behavior as test inputs.
