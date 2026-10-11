// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchSolveOutcome,
    sketch::{SketchError, SketchSolveOutcome},
};
pub(super) fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "5 cm")
        .unwrap();
    parameters
}
fn sketch_geometry(sketches: &DocumentSketches, sketch: u64) -> &Sketch2D {
    sketches.get(DocumentSketchId(sketch)).unwrap().geometry()
}
pub(super) fn solved_report(
    outcome: &DocumentSketchSolveOutcome,
    index: usize,
) -> &SketchSolveOutcome {
    outcome.report.reports[index].result.as_ref().unwrap()
}
pub(super) fn height(sketches: &DocumentSketches, sketch: u64, point: u64) -> f64 {
    sketch_geometry(sketches, sketch)
        .point(SketchEntityId(point))
        .unwrap()
        .position[1]
}
fn assert_solved_batch(outcome: &DocumentSketchSolveOutcome) {
    assert!(!outcome.report.cancelled);
    let ids: Vec<u64> = outcome
        .report
        .reports
        .iter()
        .map(|report| report.sketch.0)
        .collect();
    assert_eq!(ids, [1, 2]);
    for report in &outcome.report.reports {
        let solved = report.result.as_ref().unwrap();
        assert_eq!(solved.numerical().termination, SolveTermination::Converged);
        assert_eq!(solved.numerical().structure.degrees_of_freedom, 0);
        assert_eq!(solved.parameter_dependencies(), [1]);
        assert!(solved.conflicting_relations().is_empty());
    }
}
fn assert_solved_geometry(sketches: &DocumentSketches, width: f64) {
    assert!((height(sketches, 1, 3) - width).abs() < 1e-8);
    assert!((height(sketches, 2, 13) - width).abs() < 1e-8);
    let placed = sketches.get(DocumentSketchId(2)).unwrap();
    assert!(placed.plane().same_recipe(SketchPlane::xz()));
    assert_eq!(sketches.entity_ids(), document().entity_ids());
}
#[test]
fn ordered_batch_recomputes_parameters_without_mutating_input_identity_or_placement() {
    let original = document();
    let mut parameters = parameters();
    let mut sketches = original.clone();
    for width in [5.0, 0.0, 8.0] {
        parameters
            .set_expression(1, &format!("{width} cm"))
            .unwrap();
        let outcome = sketches.solve(&parameters, SolveOptions::default(), || false);
        assert_solved_batch(&outcome);
        sketches = outcome.candidate;
        assert_solved_geometry(&sketches, width);
    }
    assert!(original.same_recipe(&document()));
}
fn missing_dimension() -> Sketch2D {
    let mut recipe = geometry(0).recipe();
    recipe.constraints[3].relation = PlanarConstraint::Offset {
        point: SketchEntityId(3),
        line: SketchEntityId(4),
        parameter: 999,
        side: LineContactSide::Left,
        linear_diameter: false,
    };
    Sketch2D::from_recipe(&recipe).unwrap()
}
pub(super) fn duplicate_fix(position: [f64; 2]) -> Sketch2D {
    let mut first = geometry(0);
    let relation = PlanarConstraint::Fix {
        point: SketchEntityId(1),
        position,
    };
    first
        .add_constraint(SketchConstraintId(5), relation)
        .unwrap();
    first
}
fn failing_batch(first: &Sketch2D) -> DocumentSketchSolveOutcome {
    let mut sketches = document();
    sketches
        .replace_geometry(DocumentSketchId(1), first.clone())
        .unwrap();
    sketches.solve(&parameters(), SolveOptions::default(), || false)
}
fn assert_local_failure(outcome: &DocumentSketchSolveOutcome, first: &Sketch2D) {
    assert!(!outcome.report.cancelled);
    assert!(sketch_geometry(&outcome.candidate, 1).same_recipe(first));
    assert_eq!(
        solved_report(outcome, 1).numerical().termination,
        SolveTermination::Converged
    );
    assert!((height(&outcome.candidate, 2, 13) - 5.0).abs() < 1e-8);
}
#[test]
fn invalid_parameter_and_unsatisfied_constraint_keep_local_geometry_and_continue_the_batch() {
    let missing = missing_dimension();
    let outcome = failing_batch(&missing);
    assert_local_failure(&outcome, &missing);
    assert!(matches!(
        outcome.report.reports[0].result,
        Err(SketchError::Parameter { parameter: 999, .. })
    ));
    let conflicting = duplicate_fix([-1.0, 0.0]);
    let outcome = failing_batch(&conflicting);
    assert_local_failure(&outcome, &conflicting);
    let failed = solved_report(&outcome, 0);
    assert_ne!(failed.numerical().termination, SolveTermination::Converged);
    assert!(!failed.conflicting_constraints().is_empty());
}
fn first_sketch_poll_count() -> usize {
    let mut first_only = document();
    first_only.remove(DocumentSketchId(2)).unwrap();
    let mut calls = 0;
    first_only.solve(&parameters(), SolveOptions::default(), || {
        calls += 1;
        false
    });
    calls
}
fn cancelled_batch(stop_at: usize) -> DocumentSketchSolveOutcome {
    let mut calls = 0;
    document().solve(&parameters(), SolveOptions::default(), || {
        calls += 1;
        calls == stop_at
    })
}
#[test]
fn cancellation_before_during_and_after_a_converged_sketch_retains_the_complete_recipe() {
    let completed_calls = first_sketch_poll_count();
    for stop_at in [1, 2, completed_calls] {
        let outcome = cancelled_batch(stop_at);
        assert!(outcome.report.cancelled);
        assert!(document().same_recipe(&outcome.candidate));
        if stop_at == completed_calls {
            assert_eq!(outcome.report.reports.len(), 1);
            assert_eq!(
                solved_report(&outcome, 0).numerical().termination,
                SolveTermination::Converged
            );
        }
    }
}
#[test]
fn empty_documents_and_underconstraint_remain_explicit() {
    let empty = DocumentSketches::default().solve(&parameters(), SolveOptions::default(), || false);
    assert!(empty.report.reports.is_empty());
    assert!(!empty.report.cancelled);
    let mut free = Sketch2D::default();
    free.add_point(SketchEntityId(30), [2.0, 3.0]).unwrap();
    let mut sketches = document();
    sketches
        .add(DocumentSketchId(3), "Free", SketchPlane::xy(), free)
        .unwrap();
    let result = sketches.solve(&parameters(), SolveOptions::default(), || false);
    let free = solved_report(&result, 2);
    assert_eq!(free.numerical().termination, SolveTermination::Converged);
    assert_eq!(free.numerical().structure.degrees_of_freedom, 2);
}
#[test]
fn iteration_limits_and_invalid_options_do_not_adopt_unconverged_candidates() {
    for options in limited_policies() {
        let result = document().solve(&parameters(), options, || false);
        assert!(document().same_recipe(&result.candidate));
        assert_eq!(result.report.reports.len(), 2);
        for report in result.report.reports {
            match report.result {
                Ok(outcome) => assert_eq!(
                    outcome.numerical().termination,
                    SolveTermination::IterationLimit
                ),
                Err(cause) => assert!(cause.to_string().contains("max_trial_steps 0")),
            }
        }
    }
}
#[test]
fn converged_redundancy_remains_visible_in_the_document_report() {
    let outcome = failing_batch(&duplicate_fix([0.0, 0.0]));
    let first = solved_report(&outcome, 0);
    assert_eq!(first.numerical().termination, SolveTermination::Converged);
    let redundant = first
        .redundant_constraints(SolveOptions::default().rank_tolerance)
        .unwrap();
    assert!(redundant.contains(&SketchConstraintId(5)));
    assert!(first.conflicting_constraints().is_empty());
}

fn limited_policies() -> [SolveOptions; 2] {
    [
        SolveOptions {
            max_iterations: 0,
            ..Default::default()
        },
        SolveOptions {
            max_trial_steps: 0,
            ..Default::default()
        },
    ]
}
