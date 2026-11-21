// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::parameter::ParameterCollection;
fn append_distance(geometry: &mut Sketch2D, id: u64, parameter: u64) {
    let relation = PlanarConstraint::Distance {
        points: [entity(1), entity(2)],
        measure: DistanceMeasure::Aligned,
        parameter,
    };
    geometry
        .add_constraint(SketchConstraintId(id), relation)
        .unwrap();
}
fn radius_geometry(parameter: u64) -> Sketch2D {
    let mut geometry = fixture();
    geometry
        .add_constraint(
            SketchConstraintId(10),
            PlanarConstraint::Radius {
                curve: entity(6),
                parameter,
            },
        )
        .unwrap();
    geometry
}
#[test]
fn available_analysis_names_each_failed_target_and_keeps_authored_recipe_bits() {
    let mut geometry = fixture();
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(3, "healthy", MODEL_PARAM, "5 cm")
        .unwrap();
    for (id, parameter) in [(10, 1), (11, 2), (12, 3)] {
        append_distance(&mut geometry, id, parameter);
    }
    let before = geometry.recipe();
    let analysis = geometry
        .analyze_available_targets(&parameters, &[], 1e-9)
        .unwrap();
    assert_eq!(analysis.unavailable.len(), 2);
    assert_eq!(analysis.structure.degrees_of_freedom, 9);
    assert_eq!(geometry.recipe(), before);
    assert_named_unavailable(&analysis.unavailable);
}
fn assert_named_unavailable(unavailable: &[SketchError]) {
    for (failure, (id, parameter)) in unavailable.iter().zip([(10, 1), (11, 2)]) {
        assert!(
            matches!(failure, SketchError::Parameter { constraint, parameter: actual, .. } if constraint.0 == id && *actual == parameter)
        );
    }
}
#[test]
fn available_analysis_ignores_reference_targets_and_keeps_grounded_coordinates_exact() {
    let mut geometry = radius_geometry(99);
    geometry
        .set_dimension_reference(SketchConstraintId(10), true)
        .unwrap();
    let parameters = ParameterCollection::new(Default::default());
    let anchors = origin_anchor(1);
    let analysis = geometry
        .analyze_available_targets(&parameters, &anchors, 1e-9)
        .unwrap();
    assert!(analysis.unavailable.is_empty());
    assert_eq!(analysis.structure.degrees_of_freedom, 8);
    assert!(
        geometry
            .analyze_available_targets(&parameters, &anchors, -1.0)
            .is_err()
    );
}
#[test]
fn available_analysis_does_not_hide_bad_healthy_domains_or_bad_anchors() {
    let geometry = radius_geometry(1);
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "radius", MODEL_PARAM, "-1 cm")
        .unwrap();
    assert!(
        geometry
            .analyze_available_targets(&parameters, &[], 1e-9)
            .is_err()
    );
    let anchors = origin_anchor(99);
    assert!(
        geometry
            .analyze_available_targets(&parameters, &anchors, 1e-9)
            .is_err()
    );
}

fn origin_anchor(point: u64) -> [SketchPoint2D; 1] {
    [SketchPoint2D {
        id: entity(point),
        position: [0.0, 0.0],
    }]
}
