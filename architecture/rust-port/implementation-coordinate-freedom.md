# Coordinate freedom for entity inspection

This step provides the numerical dependency for sketch entity movement status.
It implements [R3 issue #3766](https://github.com/Oblikovati/Oblikovati/issues/3766).
The Go source is `model/sketch/moveable_status.go` at commit
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

A coordinate can move to first order when its unit vector has a component outside
the current constraint Jacobian's row space. A Jacobian pivot alone cannot give
that answer: two coupled coordinates can both move while one column is a pivot.
This analysis does not establish finite motion or nonlinear convergence.

`coordinate_freedoms` reuses the solver's existing normalized Jacobian rows and
column-pivoted Householder QR. It transposes the Jacobian, factors that matrix,
and measures each coordinate's squared distance from its column space.
`DenseMatrix::transpose` retains both dimensions, including empty axes. Thus a
sketch with variables and no equations retains every free coordinate; a system
with no variables returns no coordinates.

`ColumnPivotedQr::coordinate_projection_residuals` applies its retained reflectors
in forward order to compute `Q^T e_j`. The components beyond the retained rank
are perpendicular to the accepted columns. Their squared norm is the coordinate
projection residual. This uses the same rank engine and reflectors as least
squares. It adds no Gram-Schmidt implementation, normal-equation solver, or
second basis threshold.

Rank tolerance and the squared freedom threshold are separate, explicit,
validated inputs. A residual must exceed the freedom threshold. The Go
classifier uses `1e-6` for the squared residual; the model integration will retain
that policy. Rust uses normalized rows and relative pivot conditioning. The Go
implementation's absolute Gram-Schmidt cutoff is not introduced as a second
rank rule. Near a numerical rank boundary, the explicit policy determines which
columns belong to the accepted subspace.

The independent exact test matrix has columns `(1,2,3)` and `(1,2,1)`.
Its Gram matrix is `[[14,8],[8,6]]`. Jacobian's exact
`matrix.inverse.compute` operation returns the rational inverse
`[[3/10,-2/5],[-2/5,7/10]]`. Substitution in
`I - A (A^T A)^-1 A^T` gives diagonal `(4/5,1/5,0)`.
That exact inverse proves this fixture's rational calculation, not general
floating-point correctness. Production uses QR, without forming a Gram matrix.
Swapping the columns exercises pivoting and retains the same projection.

Tests also cover fixed and coupled free coordinates, full rank, dependent and
zero rows, signs, repeatability, input immutability, declared empty shapes,
near-dependence under explicit rank thresholds, and opposite row scales from
`1e-300` through `1e300`. The pinned Go movement-status tests pass independently.

The next application step must restore these free-coordinate results through
the existing grounded layout, classify each entity against one current sketch
analysis, and expose canonical entity inspection in authored order. Native drag
and pick acceptance remain required. No entity movement placeholders or API
methods are introduced by this numerical dependency.
