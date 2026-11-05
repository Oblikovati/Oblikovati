# ADR-0022: Analytic body properties at the native result boundary

Date: 2026-10-13. Status: Accepted for the Rust target design.
This extends ADR-0007 and ADR-0019 for the existing body property query.

Read physical geometry from the retained, verified native body. Use one analytic
volume integration helper for both the existing volume/centroid reports and full
centroidal inertia. This removes the private integration block in the release
probe boundary before adding the second consumer. No tessellation fallback,
second geometry engine, or application-owned eigensolver is introduced.

Keep geometric integrals at unit density in working units. The host applies the
API's centimetre-to-millimetre and density conversions, then refuses overflow.
The existing body query contract selects the part material when request density
is zero; it does not select a body material override. Explicit nonpositive density
uses one gram per cubic centimetre after default resolution, matching the Go
analysis behavior. Missing selected assets remain errors.

Principal moments use the native eigensolver and are sorted ascending with their
axes. An axis has a nonnegative largest-magnitude component, with coordinate order
breaking exact ties; signed zero is canonical. Runtime certification rejects
nonfinite integrals, nonpositive solid moments, and invalid eigenpairs. One new
numerical tolerance, 128 machine epsilons relative to the inertia matrix scale,
bounds eigenpair residual and basis normalization error. It never decides model
incidence or changes geometry. This numerical validation is the reason for a net
increase of one tolerance constant; recognizers, fallbacks, and type assertions
have no increase. The integration algorithm remains in Martensite.

Accuracy labels retain their wire meaning and are checked at the boundary. Current
planar prism integrals are analytic for every label. Future surface families must
supply their own verified analytic integration and accuracy evidence under
ADR-0007. Repeated principal moments do not imply a unique physical axis basis.
Tests must verify the eigenspace, not an arbitrary oracle sign or basis choice.
