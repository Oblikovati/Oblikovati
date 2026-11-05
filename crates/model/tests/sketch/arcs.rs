// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchRelationId;
pub(super) fn driven_arc(scale: f64, counter_clockwise: bool) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, position) in [
        (1, [0.0, 0.0]),
        (2, [3.0 * scale, 0.0]),
        (3, [0.0, 4.0 * scale]),
    ] {
        sketch.add_point(entity(id), position).unwrap();
    }
    sketch
        .add_arc(
            entity(4),
            entity(1),
            [entity(2), entity(3)],
            counter_clockwise,
        )
        .unwrap();
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
        PlanarConstraint::Horizontal {
            points: [entity(1), entity(2)],
        },
    );
    relation(
        &mut sketch,
        3,
        PlanarConstraint::Vertical {
            points: [entity(1), entity(3)],
        },
    );
    relation(
        &mut sketch,
        4,
        PlanarConstraint::Radius {
            curve: entity(4),
            parameter: 1,
        },
    );
    sketch
}
#[test]
fn radius_edits_keep_both_endpoints_circular_and_preserve_identity_and_sweep() {
    for counter_clockwise in [false, true] {
        let sketch = driven_arc(1.0, counter_clockwise);
        let original = sketch.clone();
        let mut parameters = parameters();
        let mut candidate = sketch.clone();
        for radius in [5.0, 8.0] {
            parameters
                .set_expression(1, &format!("{radius} cm"))
                .unwrap();
            let result = candidate
                .solve(&parameters, SolveOptions::default(), || false)
                .unwrap();
            assert_eq!(result.numerical().termination, SolveTermination::Converged);
            assert_eq!(result.numerical().structure.variables, 6);
            assert_eq!(result.numerical().structure.rank, 6);
            assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
            assert_eq!(result.parameter_dependencies(), [1]);
            assert_point(result.candidate(), 2, [radius, 0.0]);
            assert_point(result.candidate(), 3, [0.0, radius]);
            assert!((result.candidate().arc_radius(entity(4)).unwrap() - radius).abs() < 1e-8);
            assert_eq!(result.candidate().arcs(), original.arcs());
            assert_eq!(result.candidate().constraints(), original.constraints());
            assert!(result.conflicting_relations().is_empty());
            assert!(result.redundant_relations(1e-9).unwrap().is_empty());
            candidate = result.into_candidate();
        }
        assert_eq!(sketch, original);
    }
}
#[test]
fn arc_references_and_entity_namespace_are_validated_before_mutation() {
    let mut sketch = driven_arc(1.0, true);
    let original = sketch.clone();
    for id in [0, 1, 4] {
        assert!(
            sketch
                .add_arc(entity(id), entity(1), [entity(2), entity(3)], false)
                .is_err()
        );
    }
    assert!(
        sketch
            .add_arc(entity(8), entity(999), [entity(2), entity(3)], true)
            .unwrap_err()
            .to_string()
            .contains("999")
    );
    assert!(
        sketch
            .add_arc(entity(8), entity(1), [entity(999), entity(3)], true)
            .is_err()
    );
    assert!(sketch.add_point(entity(4), [0.0, 0.0]).is_err());
    assert!(sketch.add_line(entity(4), [entity(1), entity(2)]).is_err());
    assert!(sketch.add_circle(entity(4), entity(1), 1.0).is_err());
    assert!(
        sketch
            .arc(entity(999))
            .unwrap_err()
            .to_string()
            .contains("999")
    );
    assert!(sketch.arc_radius(entity(999)).is_err());
    assert!(
        sketch
            .add_constraint(
                constraint(8),
                PlanarConstraint::Radius {
                    curve: entity(999),
                    parameter: 1
                }
            )
            .is_err()
    );
    assert_eq!(sketch, original);
}
#[test]
fn collapsed_arc_failures_name_the_structural_source_and_retain_the_input() {
    for point in [2, 3] {
        let source = driven_arc(1.0, true);
        let mut sketch = Sketch2D::default();
        for point in source.points() {
            sketch.add_point(point.id, point.position).unwrap();
        }
        sketch
            .add_arc(entity(4), entity(1), [entity(2), entity(3)], true)
            .unwrap();
        sketch.set_position(entity(point), [0.0, 0.0]).unwrap();
        let original = sketch.clone();
        for error in [
            sketch
                .solve(&parameters(), SolveOptions::default(), || false)
                .unwrap_err(),
            sketch.analyze(&parameters(), 1e-9).unwrap_err(),
        ] {
            assert!(matches!(error, SketchError::ArcCircularity { arc, .. } if arc == entity(4)));
            assert!(error.to_string().contains("circularity"));
            assert!(std::error::Error::source(&error).is_some());
        }
        if point == 2 {
            assert!(
                sketch
                    .arc_radius(entity(4))
                    .unwrap_err()
                    .to_string()
                    .contains("radius 0.0")
            );
        }
        assert_eq!(sketch, original);
    }
    let mut sketch = driven_arc(1.0, true);
    sketch.set_position(entity(1), [-f64::MAX, 0.0]).unwrap();
    sketch.set_position(entity(2), [f64::MAX, 0.0]).unwrap();
    assert!(sketch.arc_radius(entity(4)).is_err());
}
#[test]
fn conflicts_and_redundancy_include_non_deletable_arc_circularity_sources() {
    let mut sketch = driven_arc(1.0, true);
    let limited = SolveOptions {
        max_iterations: 0,
        ..SolveOptions::default()
    };
    let result = sketch.solve(&parameters(), limited, || false).unwrap();
    assert!(
        result
            .conflicting_relations()
            .contains(&SketchRelationId::ArcCircularity(entity(4)))
    );
    assert_eq!(result.conflicting_constraints(), [constraint(4)]);
    sketch
        .add_arc(entity(5), entity(1), [entity(2), entity(3)], false)
        .unwrap();
    let result = sketch
        .solve(&parameters(), SolveOptions::default(), || false)
        .unwrap();
    assert_eq!(result.numerical().structure.variables, 6);
    assert_eq!(result.numerical().structure.redundant_equations, 1);
    assert_eq!(
        result.redundant_relations(1e-9).unwrap(),
        [
            SketchRelationId::ArcCircularity(entity(4)),
            SketchRelationId::ArcCircularity(entity(5))
        ]
    );
    assert!(result.redundant_constraints(1e-9).unwrap().is_empty());
    assert!(result.redundant_relations(-1.0).is_err());
}
#[test]
fn shared_start_end_coordinates_do_not_create_duplicate_variables() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(entity(1), [0.0, 0.0]).unwrap();
    sketch.add_point(entity(2), [3.0, 4.0]).unwrap();
    sketch
        .add_arc(entity(3), entity(1), [entity(2), entity(2)], true)
        .unwrap();
    let analysis = sketch.analyze(&parameters(), 1e-9).unwrap();
    assert_eq!(analysis.variables, 4);
    assert_eq!(analysis.rank, 0);
    assert_eq!(analysis.redundant_equations, 1);
    // Equal endpoint references are editable draft geometry, not proof of a valid profile sweep.
    assert_eq!(sketch.arc_radius(entity(3)).unwrap(), 5.0);
}
#[test]
fn arc_dof_and_circularity_solve_are_stable_across_working_unit_scales() {
    for scale in [1e-100, 1.0, 1e100] {
        let sketch = driven_arc(scale, true);
        let mut parameters = parameters();
        parameters
            .set_expression(1, &format!("{} cm", 5.0 * scale))
            .unwrap();
        assert_eq!(sketch.analyze(&parameters, 1e-9).unwrap().rank, 6);
        let result = sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap();
        assert_eq!(result.numerical().termination, SolveTermination::Converged);
        assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
        assert!((result.candidate().arc_radius(entity(4)).unwrap() / scale - 5.0).abs() < 1e-8);
        assert!(
            (result.candidate().point(entity(3)).unwrap().position[1] / scale - 5.0).abs() < 1e-8
        );
    }
}
#[test]
fn radius_parameter_errors_and_cancellation_keep_arc_definitions_intact() {
    let sketch = driven_arc(1.0, true);
    let mut parameters = parameters();
    parameters.set_expression(1, "0 cm").unwrap();
    assert!(
        sketch
            .solve(&parameters, SolveOptions::default(), || false)
            .unwrap_err()
            .to_string()
            .contains("strictly positive")
    );
    parameters.set_expression(1, "50 mm").unwrap();
    let result = sketch
        .solve(&parameters, SolveOptions::default(), || true)
        .unwrap();
    assert_eq!(result.numerical().termination, SolveTermination::Cancelled);
    assert_eq!(result.candidate(), &sketch);
}
