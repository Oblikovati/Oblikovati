# Pinned Ribbon Docking baseline

The Linux X11 run under Xvfb and Vulkan Lavapipe passed 14 of 22 scenarios,
with 144 steps passed and eight failed. It used application commit
`097ecf29674d923a70ddd4f2785c51c8178907f5`, API v0.165.0, and Martensite v0.13.1.
The scenario source was the restored snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The pinned feature SHA-256 was
`35ca58e5b39b0e08d7b10b0f9d82654e8373d34cb72e6db92ea6875525a61eaf`.
The snapshot and the feature hash were unchanged after the run.

The [Cucumber result](cucumber.json), [JUnit result](junit.xml),
[scenario arguments](scenario-arguments.json), and
[source revisions](source-revision.json) identify the run.
The [native log](native.log.gz) records the first failure in each case.
The floating and left compact group cases still need bindings and behavior
checks. The document-switch case activates the Assembly, then fails because
the left dock assertion looks for the Part-specific 3D Model tab.

The Go ribbon stays at the top on commit
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.
Seven [Go tests](go-reference.log) passed for panel fit, ribbon width and
per-tab width caching. These tests have no side or floating ribbon case.
See the [Go comparison](../2026-11-14-go-ribbon-baseline/README.md).

The 22 ledger rows define the comparison and expected results for the pinned
feature. The failed cases remain open under
[#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
The separate [published feature run](../2026-11-21-ribbon-docking-3856/README.md)
records the current source.

## Full application binding check

The [pinned binding report](pinned-application-bindings-before.json.gz)
found 1,377 unresolved step entries across the application area.
This included one ambiguous entry for a ribbon key tip. The remaining
entries span seven features. The full binding check requires zero unresolved
entries. See [#3860](https://github.com/Oblikovati/Oblikovati/issues/3860).

The [updated report](pinned-application-bindings-after.json.gz) has
1,376 undefined entries and no ambiguous entries. The focused
[Move Bodies scenario](expanded-tip.log) passed all 11 steps and the
[model-state cell scenario](model-state-text.log) passed all 19 steps.
The [separate X11 key test](expanded-tip-x11.log) also passed.

The ledger check then reported 2,416 incomplete R0 records, down from 2,438.
