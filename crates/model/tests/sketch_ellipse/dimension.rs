// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
fn driving() -> (Sketch2D, ParameterCollection<u64>) {
    let mut sketch = owned_ellipse();
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "major", USER_PARAM, "4 cm")
        .unwrap();
    sketch
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::EllipseRadius {
                ellipse: SketchEntityId(2),
                parameter: 1,
            },
        )
        .unwrap();
    (sketch, parameters)
}
#[test]
fn major_radius_relation_has_one_exact_derivative_and_removes_one_freedom() {
    let (sketch, parameters) = driving();
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 4);
    assert_eq!(
        outcome.numerical().evaluation.jacobian(),
        &[vec![0.0, 0.0, 1.0, 0.0, 0.0]]
    );
    assert_measured_major(outcome.candidate());
    assert_eq!(sketch.parameter_dependencies(), [1]);
    assert!(sketch.geometric_constraints().next().is_none());
}
fn assert_measured_major(candidate: &Sketch2D) {
    assert!(
        (candidate
            .measure_dimension(SketchConstraintId(1))
            .unwrap()
            .value
            - 4.0)
            .abs()
            < 1e-8
    );
    assert_eq!(
        candidate.ellipse(SketchEntityId(2)).unwrap().radii()[1],
        1.0
    );
    assert_eq!(
        candidate.ellipse(SketchEntityId(2)).unwrap().major_axis(),
        [0.6, 0.8]
    );
}

#[test]
fn reference_major_radius_excludes_the_target_and_keeps_measured_geometry() {
    let (mut sketch, mut parameters) = driving();
    sketch
        .set_dimension_reference(SketchConstraintId(1), true)
        .unwrap();
    parameters.set_expression(1, "missing").unwrap();
    let outcome = sketch
        .solve(&parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(outcome.numerical().structure.degrees_of_freedom, 5);
    let measured = outcome
        .candidate()
        .measure_dimension(SketchConstraintId(1))
        .unwrap();
    assert_eq!(measured.value, 3.0);
    assert!(sketch.parameter_dependencies().is_empty());
    let restored = Sketch2D::from_recipe(&sketch.recipe()).unwrap();
    assert!(restored.same_recipe(&sketch));
}

#[path = "dimension/minor.rs"]
mod minor;
