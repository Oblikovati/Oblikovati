# Extrude profile picking investigation

Date: 2026-11-13. Tracking: host [#3857](https://github.com/Oblikovati/Oblikovati/issues/3857)
and kernel [#390](https://github.com/Oblikovati/Martensite/issues/390).

The desktop Extrude dialog accepts a sketch and a current region index. It does
not yet let a user pick a closed profile in the viewport. A selected index is
resolved to authored boundary lineage before the feature is saved. An index is
not a durable identity across sketch edits.

The pinned Martensite revision `d051ff699982faba4ec2df3ee9a47e14f38dedbd`
provides `build_planar_regions` and exposes the analytic face for each region.
An experimental adapter used `BRepClassFaceClassifier` on those faces. Local
tests distinguished a rectangular ring from its inner region, rejected shared
boundary points, and checked the three origin planes and points near edges.
These examples do not establish the required classification guarantee.

The pinned classifier's intersector uses kernel precision constants and retries
intersection on an edge gap. Its public result does not provide an exact or
filtered classification certificate. The host's modeling rules require that
evidence for an incidence decision. The diagnostic classifier described in
`implementation-profile-regions.md` has the same qualification limit. The
experimental adapter was removed and is not used to select extrusion profiles.

Kernel issue #390 is already P0 and asks for exact or filtered region
classification and complete boundary incidences. A qualified picker must use
that kernel result with the resolved sketch plane and measured tolerance. It
must refuse boundary and ambiguous hits, preserve the selected boundary's
lineage, and test nested, disjoint, curved and near-degenerate regions against
an independent oracle. The desktop must then route a viewport click through the
same host operation used for Extrude and check preview, cancel and replay.
