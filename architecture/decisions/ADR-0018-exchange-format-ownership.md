# ADR-0018: Kernel-owned exchange formats

Date: 2026-10-12. Status: Accepted by the user's scope instruction.
Implementation evidence: K10 and the R8 application integration remain open.

Boundary: external CAD exchange and application integration.
Owner: RustyCADKernel owns format implementations; Oblikovati owns user workflows.

## Context and decision

The user selected [acadrust](https://github.com/hakanaktt/acadrust) for DWG
exchange and assigned all exchange format implementations to the kernel.
Use acadrust inside Martensite for DWG. The kernel owner selects its pinned
version, wraps the dependency, and tests and releases that capability.
The application does not depend directly on acadrust.

All other exchange formats follow the same ownership rule, including STEP,
IGES, DXF, mesh formats, and the legacy Inventor, OLE, and SolidWorks
translators. Do not port their parsers, writers, format-specific conversions,
or geometry algorithms in the application port. The kernel owns conversion
of format contents, units, metadata, and assembly relationships into its
exchange result contract. This clarifies the integration boundary in
[ADR-0007](ADR-0007-martensite-integration.md); its operation requirements remain.

The application owns commands, dialogs, typed automation, file access through
its storage interfaces, and calls to the released kernel exchange interface.
It applies returned content to document recipes through normal transactions,
preserves imported source resources, and shows kernel diagnostics. Application
storage of native Oblikovati recipes remains governed by
[ADR-0014](ADR-0014-documents-and-storage.md).

## Interfaces and invariants

- Public exchange additions start in Oblikovati.API. The application adapter
  consumes one kernel-owned exchange contract, with explicit format, direction,
  options, units, results, and diagnostics. Do not expose acadrust types to
  application consumers or copy public DTOs into the application.
- Missing readers or writers block their application workflows. Report a
  named unsupported capability; do not substitute an application translator,
  a Go fallback, or a native OCCT production fallback.
- Preserve imported inputs and source files on failure. The application must
  reject stale results and apply successful imports atomically.
- Format support is verified per direction and required content. A dependency
  selection or advertised upstream capability does not close K10.

## Alternatives and consequences

Porting the legacy translators inside the application would duplicate format
ownership and tie their release cycle to the host. Keep one implementation in
Martensite instead. Application integration waits for a tested kernel release;
exchange requirements remain required work, with a separate repository owner.
The kernel's internal ADR and work items are implementation prerequisites;
this application decision does not claim that they exist or are accepted.

## Validation and legacy references

The kernel tests and releases its own exchange package. Its acceptance covers
read/write fixtures, malformed inputs, units, analytic geometry, topology,
metadata, assembly structure, and explicit diagnostics for unsupported content.
Independent format oracles are required where applicable. The application
checks command and API routing, import transactions, edit/save/reopen workflows,
resource preservation, and propagation of kernel failures and diagnostics.

Use the fixed Go source and exchange inventory in the
[source map](../rust-port/source-map.md) as references for required behavior.
This decision replaces R8's earlier instruction to port recipe or domain
translators in the application. It preserves the format inventory and its
acceptance requirements. Local Linux is the current test scope; Windows and
macOS remain deferred under the user's existing instruction.
