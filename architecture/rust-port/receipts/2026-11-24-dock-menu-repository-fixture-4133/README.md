# Standalone native dock-menu fixture

Issue #4133. The standalone dock-menu test failed before any menu action
because the pinned scenario snapshot lacks
`fixtures/cad/docking-visible-solid-current.opd`. File Open left the
application without an active Part.

The old binary passed the full dock-menu test when a temporary source root
supplied the bundled Part under that name. The standalone test now opens the
same Part directly from `test-utilities/fixtures/docking-visible-solid.opd`.
The bundled fixture has SHA-256
`f631ff7cb37606f518e7d28a73c999a11362c0683d9c78e31306ba835beb2eef`.

The native test opens the Model dock menu, adds Log to the same dock region,
and closes Log while Model remains docked. The shared fixture also passes
the test that saves and reopens an extrusion with its sketch visible.
The BDD step still loads the fixture named by the scenario.
