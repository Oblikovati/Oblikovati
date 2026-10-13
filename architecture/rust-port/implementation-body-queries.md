# Current native body enumeration — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Source behavior: Go `addin/router/bodies.go`, `bodyListResult` and `bodyInfo`.

The desktop now serves the existing API-owned `body.list` contract. The file host
injects one `PartTopologyAccess` capability for enumeration and assignment
verification. A file-only host refuses enumeration without this native owner.
No DTO, method name, or geometry implementation is copied into automation.

Enumeration borrows current named results in feature replay order. Each returned
key is checked against document identity, history ownership, generation, and the
exact retained body. Failed and suppressed features have no current body to list.
Indices and default `SolidN` names describe list order; they never define identity.
Queries can prepare a missing current result, but do not edit authored history or
consume a pending window update. Rendering and queries share result storage.

Face, edge, vertex, shell, and wire counts use unique native B-rep entities. They
do not count repeated oriented uses or read tessellation. Stored body names and
legacy color-style names are read from the retained package. Invalid selected
metadata is an error. The material ID uses the current body override or part
assignment. This field reports authored intent, not resolved material properties.

Tests cover plain and holed native prisms, exact counts, stable keys after parameter
edits, failed-feature removal, query-before-render, assignment through a listed
key, undo/redo, saved display metadata after reopening, and missing owner errors.
The full gate includes the existing per-face OCCT corpus and degenerate inputs.
Kernel recognizers, tolerance constants, fallback sites, and type assertions have
no net increase. No geometry algorithm changes in this increment.

This capability covers the retained sketch-prism results. Body rename and visibility
mutation, general feature families, face enumeration, legacy-key migration,
lost-reference health, and full appearance rendering remain open. All current
published bodies are visible, matching the present scene owner. Foreground
MCPBridge acceptance and full R0–R9 acceptance remain open; this is not a full port.
