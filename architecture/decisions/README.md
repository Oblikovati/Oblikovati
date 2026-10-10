# Rust architecture decision records

The Rust application starts with a clean ADR set. This policy records the
user's decision of 2026-10-10. No application ADR has been ported or accepted
in this directory yet.

## Authority and numbering

Preexisting Go ADRs are **legacy references**. Their accepted status, numbering,
and supersession chains do not carry into the Rust application. Read them for
intent, requirements, alternatives, and lessons from the Go implementation.
Keep the original records in the Go repository, with references to the fixed
[source revision](../rust-port/source-map.md).

Start the new sequence at `ADR-0001`. Use filenames such as
`ADR-0001-<decision-title>.md`; assign numbers when records are created.
Qualify old references with the repository, commit, and legacy ADR number so
they cannot be confused with a new Rust ADR that has the same number.

Only decisions reviewed and accepted for Rust become current Rust ADRs.
Applicable legacy decisions must be rewritten for Rust ownership, lifetimes,
threading, errors, crate dependencies, native interfaces, and Martensite.
Carry forward relevant product requirements and tested behavior. Reassess Go
implementation choices and unsupported claims against the current source.

## One current ADR per architectural boundary

Consolidate legacy ADRs that describe the same architectural boundary into
one coherent Rust ADR. Organize by responsibility and interface, rather than
by old ADR number, bug, implementation wave, or feature option.

Resolve conflicting legacy decisions in the new record. State the chosen Rust
design, reasons, trade-offs, and acceptance checks together. Readers must not
need to follow a chain of legacy amendments to understand the current design.

A legacy ADR can cover more than one boundary. Map its relevant sections to
the appropriate new records. Record each decision once, then link to it from
other boundaries. Keep implementation details, incident reports, and long test
catalogues in supporting documents and issues so each ADR stays under 500 lines.

The index below will name exactly one current accepted ADR per boundary.
Drafts can change during review. If an accepted Rust decision later needs
replacement, use an explicit superseding record and update the index; retain
the historical decision. There must still be only one current authority for
that boundary.

## Migration process

R0 creates a legacy-to-Rust decision map. Each row identifies a legacy ADR or
section, its source revision, its boundary, and one disposition:

- **Adapt and consolidate:** applicable to Rust; name the target Rust record
  and the evidence needed to accept it.
- **Reference only:** useful history, with a reason it supplies no current
  Rust decision.
- **Not applicable:** specific to the old implementation; record why it is
  omitted and where any remaining product requirement is covered.

An unresolved disposition remains pending work. Do not treat a decision as
accepted because its source ADR was accepted. Removing a Go-specific design
choice does not remove an application capability from the port requirements.

Each new ADR includes its boundary and repository owner, status, context,
decision, alternatives, consequences, interfaces and invariants, validation
evidence, and legacy references. Explain what was retained, changed, combined,
or omitted. Acceptance depends on the new Rust design and evidence.

Application decisions live here. Kernel-internal decisions belong to
Martensite; API decisions belong to Oblikovati.API. A Rust application ADR
records the integration contract and links to the owning repository's record
instead of restating its internal design as a second authority.

## Consolidation example: Vulkan renderer

Create one Rust ADR titled **Vulkan renderer**. Its scope includes the
application-to-Martensite rendering interface, presentation ownership, GPU
resource lifetime and synchronization, platform capabilities, display modes,
lighting, instancing, capture, and validation. Its sections state one coherent
target design and identify capabilities that are still pending implementation.

The following legacy inputs are candidates to review at the recorded Go
revision. This list is a consolidation map, not automatic acceptance of their
old decisions.

| Legacy Go record | Input to the consolidated renderer ADR |
| --- | --- |
| ADR-0005, Vulkan 1.3 renderer | Renderer interface, device capabilities, and resource submission. |
| ADR-0014, renderer testability | Render tests and oracle strategy. |
| ADR-0019, macOS support | Runtime graphics portability; packaging and signing sections belong to the release boundary. |
| ADR-0023, viewport display modes | Required display behavior, with old backend choices reassessed. |
| ADR-0026, environment lighting | Lighting, reflections, and shadows. |
| ADR-0038, assembly render instancing | GPU instance representation; assembly model ownership belongs to its model boundary. |
| ADR-0053, OpenPBR Realistic mode | Realistic rendering requirements and validation; public appearance types remain owned by the API contract. |

Review renderer-facing sections of legacy ADR-0004 (ImGui) and ADR-0008
(cgo) when defining UI composition and native ownership. Keep the separate
UI/platform decision focused on UI and OS responsibilities, and link the
shared presentation interface to the renderer ADR.

Validate these decisions against Martensite's actual Rust renderer and the
application's requirements. The consolidated ADR must identify required
Martensite extensions, including capabilities outside OCCT. Keep capability
status in the linked implementation ledger. Do not create separate current
ADRs for each display mode or platform exception within this boundary.

## Current Rust ADR index

There are no current accepted Rust application ADRs yet. R0 starts the fresh
sequence and decision map; each later milestone supplies and verifies the
applicable consolidated records before dependent implementation. Planned
boundaries and evidence are listed in the
[port plan](../rust-port/plan.md#decisions-to-record-before-dependent-implementation).
