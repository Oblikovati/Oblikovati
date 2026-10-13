// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{SketchPlacementCreationReport, SketchPlacementFieldLock},
    sketch::SketchPlacementRecipe,
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
fn placement(curved: bool) -> SketchPlacementRecipe {
    if curved {
        return SketchPlacementRecipe::arc_slot([0.0; 2], [[10.0, 0.0], [0.0, 10.0]], 2.0, true)
            .unwrap();
    }
    SketchPlacementRecipe::straight_slot([[0.0; 2], [10.0, 0.0]], 2.0).unwrap()
}
fn committed(
    curved: bool,
    locks: &[Option<SketchPlacementFieldLock>],
) -> (DocumentEditHistory, SketchPlacementCreationReport) {
    let mut history = super::polyline::polyline_history();
    let token = history.begin("Placement").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(
                DocumentSketchId(1),
                &placement(curved),
                locks,
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    (history, report)
}
fn lock(expression: &str) -> Option<SketchPlacementFieldLock> {
    Some(SketchPlacementFieldLock {
        expression: expression.into(),
        text_point: Some([3.0, 4.0]),
    })
}
#[test]
fn constrained_slots_have_the_reference_dof_without_redundant_centerlines() {
    for curved in [false, true] {
        let (history, report) = committed(curved, &[]);
        let geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry();
        let solved = geometry
            .solve(&history.snapshot().parameters, Default::default(), || false)
            .unwrap();
        assert_eq!(solved.numerical().termination, SolveTermination::Converged);
        assert_eq!(
            solved.numerical().structure.degrees_of_freedom,
            if curved { 6 } else { 5 }
        );
        assert_eq!(solved.numerical().structure.redundant_equations, 0);
        assert_eq!(report.points.len(), if curved { 7 } else { 6 });
        assert_eq!(report.members.len(), if curved { 4 } else { 5 });
        assert_eq!(report.relations.len(), if curved { 4 } else { 5 });
        assert!(geometry.same_recipe(&Sketch2D::from_recipe(&geometry.recipe()).unwrap()));
    }
}
#[test]
fn locked_fields_create_owned_parameters_but_steering_fields_do_not() {
    for curved in [false, true] {
        let (mut history, report) = committed(
            curved,
            &[lock("10 cm"), lock("unparsed steering text"), lock("2 cm")],
        );
        assert_eq!(
            report
                .dimensions
                .iter()
                .map(|(index, _)| *index)
                .collect::<Vec<_>>(),
            vec![0, 2]
        );
        let geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry();
        let analysis = history
            .snapshot()
            .analyze_document_sketch(report.sketch, SolveOptions::default().rank_tolerance)
            .unwrap();
        assert_eq!(analysis.degrees_of_freedom, if curved { 4 } else { 3 });
        for (_, dimension) in &report.dimensions {
            assert_eq!(
                geometry.dimension_text_point(dimension.constraint).unwrap(),
                Some([3.0, 4.0])
            );
        }
        let after = history.snapshot().clone();
        history.undo().unwrap();
        assert!(
            history
                .snapshot()
                .sketches
                .get(report.sketch)
                .unwrap()
                .geometry()
                .entity_ids()
                .is_empty()
        );
        history.redo().unwrap();
        assert!(after.sketches.same_recipe(&history.snapshot().sketches));
        assert!(after.parameters.same_recipe(&history.snapshot().parameters));
    }
}
#[test]
fn duplicate_locked_fields_follow_the_document_redundancy_policy() {
    use oblikovati_api::types::OVER_CONSTRAINED_APPLY_DRIVING;
    for driving in [false, true] {
        let mut history = super::polyline::polyline_history();
        let mut recipe = placement(false);
        recipe.fields.push(recipe.fields[0].clone());
        let locks = [
            lock("10 cm"),
            None,
            None,
            lock(if driving { "10 cm" } else { "20 cm" }),
        ];
        let token = history.begin("Redundancy policy").unwrap();
        let report = history
            .edit(&token, |snapshot, ids| {
                if driving {
                    let mut settings = snapshot.effective_sketch_settings();
                    settings.over_constrained_behavior = OVER_CONSTRAINED_APPLY_DRIVING;
                    snapshot.sketch_settings = Some(settings);
                }
                snapshot.create_sketch_placement(
                    DocumentSketchId(1),
                    &recipe,
                    &locks,
                    Default::default(),
                    ids,
                )
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        let geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry();
        let second = &report.dimensions[1].1;
        assert_eq!(
            geometry.dimension_is_reference(second.constraint).unwrap(),
            !driving
        );
        assert_eq!(
            history
                .snapshot()
                .analyze_document_sketch(report.sketch, SolveOptions::default().rank_tolerance)
                .unwrap()
                .redundant_equations,
            usize::from(driving)
        );
        assert!(
            (geometry
                .measure_dimension(report.dimensions[0].1.constraint)
                .unwrap()
                .value
                - 10.0)
                .abs()
                < 1e-7
        );
    }
}
#[test]
fn late_field_failure_discards_geometry_relations_and_earlier_parameter_sources() {
    use std::error::Error;
    let mut history = super::polyline::polyline_history();
    let before = history.snapshot().clone();
    let token = history.begin("Bad field").unwrap();
    let error = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(
                DocumentSketchId(1),
                &placement(false),
                &[lock("10 cm"), None, lock("not_a_width")],
                Default::default(),
                ids,
            )
        })
        .unwrap_err();
    let EditError::Mutation(cause) = error else {
        panic!("expected field mutation failure")
    };
    assert!(cause.to_string().contains("Width"), "{cause:?}");
    assert!(cause.source().is_some());
    assert!(before.sketches.same_recipe(&history.snapshot().sketches));
    assert!(
        before
            .parameters
            .same_recipe(&history.snapshot().parameters)
    );
    assert_eq!(history.sketch_reservations().entity, 11);
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(
                DocumentSketchId(1),
                &placement(false),
                &[lock("10 cm")],
                Default::default(),
                ids,
            )
        })
        .unwrap();
    assert!(report.dimensions[0].1.parameter > 2);
}
#[test]
fn malformed_recipe_indices_and_field_values_never_publish_partial_geometry() {
    use oblikovati_model::sketch::{
        SketchPathMember, SketchPlacementBinding, SketchPlacementRelation,
    };
    let original = placement(false);
    let mut bad = Vec::new();
    let mut recipe = original.clone();
    recipe.points[0][0] = f64::NAN;
    bad.push(recipe);
    let mut recipe = original.clone();
    recipe.members[0].path = SketchPathMember::Line([99, 1]);
    bad.push(recipe);
    let mut recipe = original.clone();
    recipe.relations[0] = SketchPlacementRelation::EqualRadius([1, 99]);
    bad.push(recipe);
    let mut recipe = original.clone();
    recipe.fields[0].binding = SketchPlacementBinding::Distance([0, 99]);
    bad.push(recipe);
    let mut recipe = original.clone();
    recipe.fields[0].binding = SketchPlacementBinding::Diameter(99);
    bad.push(recipe);
    let mut recipe = original.clone();
    recipe.fields[0].value = f64::NAN;
    bad.push(recipe);
    let mut recipe = original;
    recipe.fields[0].witness[0][0] = f64::INFINITY;
    bad.push(recipe);
    for recipe in bad {
        let mut history = super::polyline::polyline_history();
        let before = history.snapshot().clone();
        let token = history.begin("Bad recipe").unwrap();
        assert!(
            history
                .edit(&token, |snapshot, ids| snapshot.create_sketch_placement(
                    DocumentSketchId(1),
                    &recipe,
                    &[],
                    Default::default(),
                    ids
                ))
                .is_err()
        );
        assert!(before.sketches.same_recipe(&history.snapshot().sketches));
        assert!(
            before
                .parameters
                .same_recipe(&history.snapshot().parameters)
        );
    }
}
#[test]
fn recipe_entity_and_constraint_exhaustion_leave_the_original_document_intact() {
    use std::error::Error;
    for (entity, constraint) in [(u64::MAX - 3, 0), (0, u64::MAX - 2)] {
        let mut history = super::polyline::polyline_history()
            .with_sketch_reservations(SketchReservationMarks {
                sketch: 1,
                entity,
                constraints: [(1, constraint)].into(),
                ..Default::default()
            })
            .unwrap();
        let before = history.snapshot().clone();
        let token = history.begin("Exhaust recipe").unwrap();
        let error = history
            .edit(&token, |snapshot, ids| {
                snapshot.create_sketch_placement(
                    DocumentSketchId(1),
                    &placement(false),
                    &[],
                    Default::default(),
                    ids,
                )
            })
            .unwrap_err();
        let EditError::Mutation(cause) = error else {
            panic!("expected exhausted recipe reservation")
        };
        assert!(cause.source().is_some());
        assert!(matches!(
            cause,
            oblikovati_model::history::SketchPlacementCreationError::Primitive(_)
                | oblikovati_model::history::SketchPlacementCreationError::Constraint(_)
        ));
        assert!(before.sketches.same_recipe(&history.snapshot().sketches));
    }
}
#[test]
fn slot_recipes_remain_fully_constrained_through_length_radius_and_width_edits() {
    use oblikovati_model::sketch::{PlanarConstraint, SketchConstraintId};
    for curved in [false, true] {
        let (history, report) = committed(curved, &[lock("10 cm"), None, lock("2 cm")]);
        let mut parameters = history.snapshot().parameters.clone();
        let mut geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .clone();
        let (center, end) = if curved { (0, 5) } else { (5, 4) };
        geometry
            .add_constraint(
                SketchConstraintId(1000),
                PlanarConstraint::Fix {
                    point: report.points[center],
                    position: [0.0; 2],
                },
            )
            .unwrap();
        geometry
            .add_constraint(
                SketchConstraintId(1001),
                PlanarConstraint::Horizontal {
                    points: [report.points[center], report.points[end]],
                },
            )
            .unwrap();
        if curved {
            parameters
                .insert_expression(9000, "sweep", oblikovati_api::types::USER_PARAM, "90 deg")
                .unwrap();
            geometry
                .add_three_point_angle(
                    SketchConstraintId(1002),
                    report.points[0],
                    [report.points[5], report.points[6]],
                    9000,
                )
                .unwrap();
        }
        for (size, width) in [(10.0, 2.0), (14.0, 3.0), (7.0, 1.0)] {
            parameters
                .set_expression(report.dimensions[0].1.parameter, &format!("{size} cm"))
                .unwrap();
            parameters
                .set_expression(report.dimensions[1].1.parameter, &format!("{width} cm"))
                .unwrap();
            let solved = geometry
                .solve(&parameters, Default::default(), || false)
                .unwrap();
            assert_eq!(solved.numerical().termination, SolveTermination::Converged);
            assert_eq!(solved.numerical().structure.degrees_of_freedom, 0);
            assert_eq!(solved.numerical().structure.redundant_equations, 0);
            geometry = solved.into_candidate();
            verify_dimensioned_slot(&geometry, &report, curved, size, width);
        }
    }
}
fn verify_dimensioned_slot(
    geometry: &Sketch2D,
    report: &SketchPlacementCreationReport,
    curved: bool,
    size: f64,
    width: f64,
) {
    let half = width / 2.0;
    let expected = if curved {
        vec![
            [0.0; 2],
            [size + half, 0.0],
            [0.0, size + half],
            [0.0, size - half],
            [size - half, 0.0],
            [size, 0.0],
            [0.0, size],
        ]
    } else {
        vec![
            [0.0, half],
            [size, half],
            [size, -half],
            [0.0, -half],
            [size, 0.0],
            [0.0; 2],
        ]
    };
    for (id, expected) in report.points.iter().zip(expected) {
        let actual = geometry.point(*id).unwrap().position;
        for axis in 0..2 {
            assert!(
                (actual[axis] - expected[axis]).abs() < 1e-7,
                "{id:?}: {actual:?}, expected {expected:?}"
            );
        }
    }
}
#[test]
fn driving_policy_does_not_accept_an_unavailable_locked_field() {
    let mut history = super::polyline::polyline_history();
    let before = history.snapshot().clone();
    let token = history.begin("Invalid driving field").unwrap();
    let result = history.edit(&token, |snapshot, ids| {
        let mut settings = snapshot.effective_sketch_settings();
        settings.over_constrained_behavior = oblikovati_api::types::OVER_CONSTRAINED_APPLY_DRIVING;
        snapshot.sketch_settings = Some(settings);
        snapshot.create_sketch_placement(
            DocumentSketchId(1),
            &placement(false),
            &[lock("missing_length")],
            Default::default(),
            ids,
        )
    });
    assert!(result.is_err());
    assert!(before.sketches.same_recipe(&history.snapshot().sketches));
    assert!(
        before
            .parameters
            .same_recipe(&history.snapshot().parameters)
    );
}
