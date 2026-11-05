// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{SketchPlacementCreationReport, SketchPlacementFieldLock},
    sketch::{
        PlanarConstraint, SketchConstraintId, SketchPlacementRecipe, SketchPlacementRelation,
    },
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
fn rectangle(kind: usize) -> SketchPlacementRecipe {
    match kind {
        0 => SketchPlacementRecipe::rectangle([[0., 0.], [10., 5.]]),
        1 => SketchPlacementRecipe::center_rectangle([0., 0.], [5., 2.5]),
        _ => SketchPlacementRecipe::three_point_rectangle([[0., 0.], [10., 0.]], [4., 5.]),
    }
    .unwrap()
}
fn rectangle_history(
    kind: usize,
    locked: bool,
) -> (DocumentEditHistory, SketchPlacementCreationReport) {
    let mut history = super::polyline::polyline_history();
    let recipe = rectangle(kind);
    let mut locks = vec![
        super::placement::lock("10 cm"),
        super::placement::lock("5 cm"),
    ];
    if kind == 2 {
        locks.insert(1, None);
    }
    let token = history.begin("Rectangle").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(
                DocumentSketchId(1),
                &recipe,
                if locked { &locks } else { &[] },
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    (history, report)
}
#[test]
fn rectangle_relations_have_go_rank_and_no_redundant_equations() {
    for kind in 0..3 {
        let (mut history, report) = rectangle_history(kind, false);
        let snapshot = history.snapshot().clone();
        let geometry = snapshot.sketches.get(report.sketch).unwrap().geometry();
        let solved = geometry
            .solve(&snapshot.parameters, Default::default(), || false)
            .unwrap();
        assert_eq!(solved.numerical().termination, SolveTermination::Converged);
        assert_eq!(
            solved.numerical().structure.degrees_of_freedom,
            if kind == 2 { 5 } else { 4 }
        );
        assert_eq!(solved.numerical().structure.redundant_equations, 0);
        assert_eq!(report.members.len(), if kind == 1 { 6 } else { 4 });
        assert_eq!(report.points.len(), if kind == 1 { 5 } else { 4 });
        assert!(geometry.same_recipe(&Sketch2D::from_recipe(&geometry.recipe()).unwrap()));
        history.undo().unwrap();
        assert!(
            history
                .snapshot()
                .sketches
                .get(report.sketch)
                .unwrap()
                .geometry()
                .points()
                .is_empty()
        );
        history.redo().unwrap();
        assert!(history.snapshot().sketches.same_recipe(&snapshot.sketches));
    }
}
#[test]
fn fully_constrained_rectangles_follow_dimension_edits_and_keep_every_corner() {
    for kind in 0..3 {
        let (history, report) = rectangle_history(kind, true);
        let mut parameters = history.snapshot().parameters.clone();
        let mut geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .clone();
        let anchor = if kind == 1 { 4 } else { 0 };
        geometry
            .add_constraint(
                SketchConstraintId(1000),
                PlanarConstraint::Fix {
                    point: report.points[anchor],
                    position: [0., 0.],
                },
            )
            .unwrap();
        if kind == 2 {
            geometry
                .add_constraint(
                    SketchConstraintId(1001),
                    PlanarConstraint::LineHorizontal {
                        line: report.members[0],
                    },
                )
                .unwrap();
        }
        assert_eq!(report.dimensions.len(), 2);
        let width_name = parameters
            .get(report.dimensions[0].1.parameter)
            .unwrap()
            .name
            .clone();
        parameters
            .set_expression(
                report.dimensions[1].1.parameter,
                &format!("{width_name} / 2"),
            )
            .unwrap();
        for (width, height) in [(10., 5.), (14., 7.), (6., 3.)] {
            parameters
                .set_expression(report.dimensions[0].1.parameter, &format!("{width} cm"))
                .unwrap();

            let solved = geometry
                .solve(&parameters, Default::default(), || false)
                .unwrap();
            assert_eq!(solved.numerical().termination, SolveTermination::Converged);
            assert_eq!(solved.numerical().structure.degrees_of_freedom, 0);
            assert_eq!(solved.numerical().structure.redundant_equations, 0);
            geometry = solved.into_candidate();
            verify_corners(&geometry, &report, kind == 1, width, height);
        }
    }
}
fn verify_corners(
    geometry: &Sketch2D,
    report: &SketchPlacementCreationReport,
    centered: bool,
    width: f64,
    height: f64,
) {
    let [x, y] = if centered {
        [-width / 2., -height / 2.]
    } else {
        [0., 0.]
    };
    let expected = [
        [x, y],
        [x + width, y],
        [x + width, y + height],
        [x, y + height],
    ];
    for (id, want) in report.points.iter().zip(expected) {
        let actual = geometry.point(*id).unwrap().position;
        for i in 0..2 {
            assert!((actual[i] - want[i]).abs() < 1e-7, "{actual:?} != {want:?}");
        }
    }
    if centered {
        // The fix is a solved equation; check achieved coordinate precision as for the corners.
        assert!(
            geometry
                .point(report.points[4])
                .unwrap()
                .position
                .iter()
                .all(|value| value.abs() < 1e-7)
        );
    }
}
#[test]
fn bad_midpoint_indices_and_late_locked_fields_leave_no_partial_rectangle() {
    for failure in 0..3 {
        let mut history = super::polyline::polyline_history();
        let before = history.snapshot().clone();
        let mut recipe = rectangle(1);
        if failure < 2 {
            recipe.relations.push(SketchPlacementRelation::Midpoint {
                point: if failure == 0 { 99 } else { 4 },
                line: if failure == 1 { 99 } else { 4 },
            });
        }
        let locks: Vec<Option<SketchPlacementFieldLock>> = if failure == 2 {
            vec![
                super::placement::lock("10 cm"),
                super::placement::lock("missing_parameter"),
            ]
        } else {
            vec![]
        };
        let token = history.begin("Refused rectangle").unwrap();
        assert!(
            history
                .edit(&token, |snapshot, ids| snapshot.create_sketch_placement(
                    DocumentSketchId(1),
                    &recipe,
                    &locks,
                    SolveOptions::default(),
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
        history.abort(&token).unwrap();
    }
}
