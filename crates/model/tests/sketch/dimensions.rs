// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::error::Error;

#[test]
fn oriented_distances_retain_the_warm_start_branch_and_report_one_remaining_dof() {
    for (measure, expected) in [
        (DistanceMeasure::Aligned, [-3.0, -4.0]),
        (DistanceMeasure::Horizontal, [-5.0, -4.0]),
        (DistanceMeasure::Vertical, [-3.0, -5.0]),
    ] {
        let mut sketch = Sketch2D::default();
        sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
        sketch.add_point(entity(2), [-3.0, -4.0]).unwrap();
        relation(
            &mut sketch,
            1,
            PlanarConstraint::Fix {
                point: entity(1),
                position: [0.0, 0.0],
            },
        );
        relation(
            &mut sketch,
            2,
            PlanarConstraint::Distance {
                points: [entity(1), entity(2)],
                measure,
                parameter: 1,
            },
        );
        let solved = sketch
            .solve(&parameters(), SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(solved.numerical().termination, SolveTermination::Converged);
        assert_eq!(solved.numerical().structure.degrees_of_freedom, 1);
        assert_point(solved.candidate(), 2, expected);
    }
}
#[test]
fn dimensions_read_selected_engineering_values_and_keep_first_use_dependencies() {
    let mut sketch = rectangle();
    relation(
        &mut sketch,
        99,
        PlanarConstraint::Distance {
            points: [entity(1), entity(2)],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
    );
    let mut parameters = parameters();
    parameters.set_deviation(1, 0.5, -0.2).unwrap();
    parameters
        .set_model_value_type(1, oblikovati_api::types::MODEL_VALUE_UPPER)
        .unwrap();
    let solved = sketch
        .solve(&parameters, SolveOptions::default(), || false)
        .unwrap();
    assert_rectangle(solved.candidate(), 5.5, 2.5);
    assert_eq!(solved.parameter_dependencies(), [1, 2]);
    assert_eq!(
        parameters.get(1).unwrap().current_value().unwrap().value,
        5.0
    );
}
#[test]
fn numerical_and_parameter_errors_keep_their_typed_causes() {
    let sketch = rectangle();
    let options = SolveOptions {
        rank_tolerance: f64::NAN,
        ..SolveOptions::default()
    };
    let error = sketch.solve(&parameters(), options, || false).unwrap_err();
    assert!(matches!(error, SketchError::Numerical(_)));
    assert!(error.source().is_some());
    let error = sketch
        .solve(
            &ParameterCollection::new(UnitsOfMeasure::default()),
            SolveOptions::default(),
            || false,
        )
        .unwrap_err();
    assert!(error.source().is_some());
    let error = sketch.point(entity(999)).unwrap_err();
    assert!(error.source().is_none());
    assert!(sketch.analyze(&parameters(), f64::NAN).is_err());
}
