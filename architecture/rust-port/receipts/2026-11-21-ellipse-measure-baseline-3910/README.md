# Ellipse Measure selection baseline

The published Ellipse run at `531d5b938` reached Measure in ellipse-05
after creating the rotated ellipse and restoring it with Redo. The ellipse
was visible and the Measure panel was open. The pointer selection left the
panel empty. The second full diagnostic reproduced the same failure.

The application integration test
`rotated_ellipse_pick_matches_the_independent_occt_minimum`
reproduced the same kernel problem with Martensite v0.13.1. It expected
the independent OCCT distance of `0.019024451883025126` model units.
Martensite returned a boundary witness at distance `0.3021561463000778`.
The earlier pinned diagnostic records the projected pointer and its expected
interior witness.

The scenario stays blocked on
[Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).
