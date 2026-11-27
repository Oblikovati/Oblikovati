# ADR-0023: Compose analytic body integrals for part analysis

Date: 2026-10-13. Status: Accepted for the Rust target design.
This extends ADR-0022 for the existing whole-part mass query.

Use one native integration pipeline for a body and a collection of bodies.
Replace the separate single-body integration sequence with the nonempty-system
pipeline before adding the whole-part caller. Keep the existing shared unit
conversion, density selection, and principal-inertia certification.

Enumerate current successful bodies in feature replay order and verify every
body's current semantic name before integration. Compose their unit-density
integrals with Martensite's `GProps::add`, which applies the parallel-axis theorem.
Seed the system with its first body so the one-body result remains bit-identical.
Do not form a Boolean union: the Go analysis sums each body's volume and exposed
area even when separate bodies overlap. This preserves the analysis contract.

An empty collection has no solid integral. The host emits the API's zero geometry
and mass, resolved density, and identity principal axes. Failed features have no
current body; their health remains available through the existing feature path.
This query does not invent a last-good solid or change authored history.

No recognizer, numerical tolerance, geometric fallback, or native type assertion
is added. OCCT aggregate fixtures must test translated systems and off-diagonal
inertia at multiple scales; whole-body sums alone are insufficient. The existing
per-face corpus remains a gate for each native body construction.
