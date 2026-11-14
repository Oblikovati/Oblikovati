# ADR-0034: Identity supplies collinear incidence

Status: Accepted for the Rust target. Date: 2026-11-06.
Tracking: [#3837](https://github.com/Oblikovati/Oblikovati/issues/3837).

## Context

Two disjoint lines require direction and offset equations for collinearity.
Shared point identity already supplies incidence. Evaluating that incidence as
a second floating-point equation can leave a cancelled derivative that row
normalization promotes into a false rank decision. Native tool qualification
exposed DOF four instead of five for three connected free points.

## Decision

Equation preparation classifies shared endpoint identity once. Zero shared
endpoints retain the original direction and offset formulas bit for bit. One
shared endpoint retains only the direction equation. Two shared endpoints
already define the same support and contribute no equation. Direction validity
is still checked, including for identity-only relations. Distinct point IDs are
never merged because their coordinates coincide. The stored authored relation,
source identity and persistence encoding remain Collinear.

This deletes redundant incidence rows before AD and rank analysis. It adds no
tolerance, geometry recognizer, coordinate perturbation or second solver path.
The classifier uses identity; geometric coincidence remains a separate relation.

## Evidence

Regression cases cover zero, one and two shared endpoints, both endpoint orders
and three scales. Existing disjoint-line Go derivative fixtures remain hard
assertions. Solving, identity retention and coincident independent points must
retain their own equations.
