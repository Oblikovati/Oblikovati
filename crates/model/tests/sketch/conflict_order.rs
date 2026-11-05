// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    sketch::{PlanarConstraint, SketchEntityId},
    units::UnitsOfMeasure,
};
fn conflicting_fix_outcome() -> SketchSolveOutcome {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0.5, 0.]).unwrap();
    for (id, position) in [(71, [0., 0.]), (27, [1., 0.])] {
        sketch
            .add_constraint(
                SketchConstraintId(id),
                PlanarConstraint::Fix {
                    point: SketchEntityId(1),
                    position,
                },
            )
            .unwrap();
    }
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap()
}
#[test]
fn conflict_reporting_is_independent_of_numerical_severity_order() {
    let mut result = conflicting_fix_outcome();
    let candidate = result.candidate().points().to_vec();
    let values = result.numerical().values.clone();
    for ranking in [[0, 1], [1, 0]] {
        result.numerical.conflicts = ranking.to_vec();
        assert_eq!(
            result.conflicting_constraints(),
            [SketchConstraintId(71), SketchConstraintId(27)]
        );
        assert_eq!(result.numerical().conflicts, ranking);
        assert_eq!(result.numerical().values, values);
        assert_eq!(result.candidate().points(), candidate);
    }
}
