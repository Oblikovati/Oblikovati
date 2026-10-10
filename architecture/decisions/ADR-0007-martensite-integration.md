# ADR-0007: Martensite operation integration

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: K01-K11 remain open until tested for Oblikovati.

Boundary: feature evaluation to kernel operations.
Owner: the application adapter; algorithms and internal kernel decisions belong
to RustyCADKernel. This record sets requirements on results, not a second kernel
architecture or a claim that Martensite already meets them.

## Context

The Go ADRs contain years of numerical work, competing Boolean paths, retired
fallbacks, and case-specific fixes. Copying their implementation sequence would
recreate that debt. OCCT coverage also does not cover every capability that the
Go kernel supplies to the application.

## Decision

Use one model-owned evaluation interface implemented by the Martensite adapter.
It accepts immutable recipes, revision-scoped shape inputs, explicit units,
resolution, and a versioned options value with stable defaults. It returns
validated geometry, operation history, achieved tolerances, and diagnostics.
Keep conversion and orchestration here; implement geometry in Martensite.

Classify each required behavior, including options, history, and failure modes:

| Evidence | Action |
| --- | --- |
| Martensite supplies equivalent behavior | Reuse it and prove the application contract. |
| OCCT supplies it, Martensite does not | Complete and validate the Martensite OCCT port. |
| The Go kernel supplies it and OCCT does not | Port and adapt it into Martensite as a required extension. |

An uncertain mapping remains open. A missing same-named method does not prove
absence from OCCT. For every extension preserve Go source revision, tests,
fixtures, application consumers, the owning Martensite decision/work item,
and the old implementation it replaces. Track extension acceptance separately
from OCCT parity. Absence from OCCT never removes an application requirement.

## Interfaces and invariants

### Required operation contract

- One general pipeline per operation; geometric fast paths belong inside it
  and must preserve topology, history, and tolerances. Dispatch classifies once.
  Unsupported cases return named declines before building invalid output.
- Every topological decision uses an exact or filtered predicate. Incidences
  are decided once and reused. Metric tolerances cannot decide topology by
  arbitrary epsilon or repair it by moving output geometry.
- One explicit resolution policy distinguishes same-computation weld,
  independent-source sew, and angular-to-length conversion. Report measured
  achieved tolerance, enforce its cap and repair-ratio bound, and keep budgets
  from loosening as operations compose. Keep meshing/fit tolerance separate.
  Coincidence classes must be transitive. Every threshold needs a documented
  resolution source or justified tolerance class, not an unexplained literal.
- Validate every successful public operation in ordered levels: topology,
  geometry/topology consistency, self-intersection, and tolerance consistency.
  An invalid body is not a successful result. Local failure identifies the
  faulty entity and any diagnostic partial result.
- Generated/modified/deleted history and preserved input identity are required
  by [ADR-0005](ADR-0005-identity-and-references.md). Healing is an explicit
  operation on a copy, with an identity remap.
- Tessellation is downstream. Modeling, containment, mass properties, section
  curves, exact drawing HLR, and persistent identity use analytic B-rep data.
  Display triangles cannot replace missing analytic functionality.
- Kernel faces retain explicit parametric trim, pcurves, and orientation.
  Periodic regions and shared edge subdivisions must stay consistent through
  operations; the adapter cannot recover missing topology from sampled loops.
- Root and branch choices need geometric certificates. A sampled tally or a
  whole-body volume match alone cannot certify a correct branch or face set.
- Deterministic output requires total orders and a verified arithmetic policy.
  Port the numeric intent of legacy ADR-0064, not Go casts as alleged Rust
  rounding barriers. Test the actual Rust compiler and target arithmetic.

### Kernel-owned work

Boolean, blend, projection, section, tessellation, validation, and healing
decisions must be consolidated in the owning Martensite boundaries when those
operations change. Reuse its geometry, topology, and history interfaces.
Required Go behavior includes any verified extension of those interfaces;
the extension list is not limited to new operation names.

The required shared blend design covers fillet, chamfer, and draft through
spine, section functional, marcher, and corner fill. Record its adaptation and
the algorithms it replaces in the owning Martensite ADR before implementation.
Do not recreate old special-case blend providers in the application. A temporary
kernel migration must name the deletion issue and corpus gate, last no more than one milestone,
and report changes in recognizers, tolerance constants, fallback sites, and
type assertions. Any positive net requires a justified owning-kernel ADR.

Martensite's current port plan does not itself approve these application
extensions. Required extension decisions and work items there are still
implementation prerequisites. Link them from the
[K11 ledger](../rust-port/martensite.md#required-extensions-from-the-go-kernel).

## Alternatives and consequences

A Go-kernel FFI fallback, native OCCT fallback, private application algorithm,
or tessellated modeling rescue would create another production kernel path.
Reject them. A missing kernel capability blocks its dependent feature. This
cost is explicit and prevents a visually plausible but incorrect port.

## Legacy disposition

Adapt the applicable contracts from Oblikovati ADR-0027, 0030, 0042, 0045-0052,
0055-0066 at the [fixed revision](legacy-map.md#source-baseline). Keep their
case histories and fixtures as input to kernel work. Do not copy per-family
handlers, tolerance literals, fixed sample counts, or old shipping claims.
The mesh-reconstruction cutover of 0057 was reversed; its multi-parent naming
requirement is retained in ADR-0005. Old CSG rescue permissions are not retained.

Legacy [issue #3555](https://github.com/Oblikovati/Oblikovati/issues/3555)
records five corrections to ADR-0061. Its long implementation log remains
reference material, not a current capability ledger. The two-engine blend
history in [#2200](https://github.com/Oblikovati/Oblikovati/issues/2200) is a
regression risk to avoid, not a Rust architecture to reproduce.

## Validation required

R1 starts with shape-isolation, units, primitive/extrusion, analytic properties,
and provenance proofs. Each later operation needs invariant tests, hard
OCCT/SolveSpace comparisons where applicable, and generated degenerate inputs.
Check per-face type, area, loops, adjacency, orientation, and reference meaning.
For behavior without an OCCT equivalent, use independent analytic results or
proved invariants; agreement with the Go code alone is insufficient.

Use the pinned kernel source and receipts listed in the
[migration evidence](legacy-map.md#source-baseline). Clone/shared-handle
semantics and the limited `rustycad-kernel` facade remain integration checks.
This migration ran no kernel acceptance tests and closes no K01-K11 gate.
