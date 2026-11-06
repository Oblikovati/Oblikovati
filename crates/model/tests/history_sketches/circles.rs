// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{SketchPlacementCreationReport, SketchPlacementFieldLock},
    parameter::ParameterCollection,
    sketch::{
        PlanarConstraint, SketchConstraintId, SketchPlacementGeometry, SketchPlacementRecipe,
    },
};
use oblikovati_solve::nonlinear::SolveTermination;
fn circle_history(
    locks: &[Option<SketchPlacementFieldLock>],
) -> (DocumentEditHistory, SketchPlacementCreationReport) {
    let mut history = super::polyline::polyline_history();
    let token = history.begin("Circle").unwrap();
    let recipe = SketchPlacementRecipe::circle([2., 3.], 5.).unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_placement(
                DocumentSketchId(1),
                &recipe,
                locks,
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    (history, report)
}
fn circle_geometry(history: &DocumentEditHistory) -> &Sketch2D {
    history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
}
#[test]
fn circle_commit_round_trips_and_undo_removes_the_whole_graph() {
    let (mut history, report) = circle_history(&[]);
    let before = history.snapshot().clone();
    let geometry = circle_geometry(&history);
    assert_eq!(report.points.len(), 1);
    assert_eq!(report.members.len(), 1);
    assert!(report.relations.is_empty());
    assert_eq!(
        geometry.circle(report.members[0]).unwrap().center,
        report.points[0]
    );
    assert!(geometry.same_recipe(&Sketch2D::from_recipe(&geometry.recipe()).unwrap()));
    history.undo().unwrap();
    assert!(circle_geometry(&history).points().is_empty());
    history.redo().unwrap();
    assert!(history.snapshot().sketches.same_recipe(&before.sketches));
}
#[test]
fn locked_diameter_and_fixed_center_solve_without_redundant_equations() {
    let (history, report) = circle_history(&[super::placement::lock("2 * 5 cm")]);
    let mut geometry = circle_geometry(&history).clone();
    let fixed = PlanarConstraint::Fix {
        point: report.points[0],
        position: [2., 3.],
    };
    geometry
        .add_constraint(SketchConstraintId(1000), fixed)
        .unwrap();
    let mut parameters = history.snapshot().parameters.clone();
    assert_eq!(report.dimensions.len(), 1);
    for diameter in [10., 20., 0.25] {
        parameters
            .set_expression(report.dimensions[0].1.parameter, &format!("{diameter} cm"))
            .unwrap();
        geometry = solve_fixed_circle(&geometry, &parameters, &report, diameter);
    }
}
fn solve_fixed_circle(
    geometry: &Sketch2D,
    parameters: &ParameterCollection<u64>,
    report: &SketchPlacementCreationReport,
    diameter: f64,
) -> Sketch2D {
    let solved = geometry
        .solve(parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(solved.numerical().termination, SolveTermination::Converged);
    assert_eq!(solved.numerical().structure.degrees_of_freedom, 0);
    assert_eq!(solved.numerical().structure.redundant_equations, 0);
    let candidate = solved.into_candidate();
    assert!((candidate.circle(report.members[0]).unwrap().radius - diameter / 2.).abs() < 1e-8);
    assert_eq!(
        candidate.point(report.points[0]).unwrap().position,
        [2., 3.]
    );
    candidate
}
#[test]
fn malformed_circle_indices_radii_and_late_fields_leave_no_partial_graph() {
    for failure in 0..3 {
        verify_refused_circle(failure);
    }
}
fn bad_circle_recipe(failure: usize) -> SketchPlacementRecipe {
    let mut recipe = SketchPlacementRecipe::circle([0., 0.], 5.).unwrap();
    recipe.members[0].geometry = SketchPlacementGeometry::Circle {
        center: if failure == 0 { 99 } else { 0 },
        radius: if failure == 1 { -1. } else { 5. },
    };
    recipe
}
fn verify_refused_circle(failure: usize) {
    let mut history = super::polyline::polyline_history();
    let before = history.snapshot().clone();
    let recipe = bad_circle_recipe(failure);
    let locks = if failure == 2 {
        vec![super::placement::lock("missing_parameter")]
    } else {
        vec![]
    };
    let token = history.begin("Refused circle").unwrap();
    let refused = history.edit(&token, |snapshot, ids| {
        snapshot.create_sketch_placement(
            DocumentSketchId(1),
            &recipe,
            &locks,
            Default::default(),
            ids,
        )
    });
    assert!(refused.is_err());
    assert!(before.sketches.same_recipe(&history.snapshot().sketches));
    assert!(
        before
            .parameters
            .same_recipe(&history.snapshot().parameters)
    );
    history.abort(&token).unwrap();
}
