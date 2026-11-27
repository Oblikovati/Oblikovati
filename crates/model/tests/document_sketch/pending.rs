// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::{height, parameters, solved_report};
use super::*;
use oblikovati_model::document_sketch::DocumentSketchSolveOutcome;
fn separate_readers() -> (DocumentSketches, ParameterCollection<u64>) {
    let mut sketches = document();
    let mut parameters = parameters();
    parameters
        .insert_expression(2, "height", USER_PARAM, "7 cm")
        .unwrap();
    let mut recipe = geometry(10).recipe();
    let PlanarConstraint::Offset { parameter, .. } = &mut recipe.constraints[3].relation else {
        panic!("expected offset fixture");
    };
    *parameter = 2;
    sketches
        .replace_geometry(DocumentSketchId(2), Sketch2D::from_recipe(&recipe).unwrap())
        .unwrap();
    (sketches, parameters)
}
fn attempted(outcome: &DocumentSketchSolveOutcome) -> Vec<u64> {
    outcome
        .report
        .reports
        .iter()
        .map(|report| report.sketch.0)
        .collect()
}
#[test]
fn initial_verification_then_changed_reader_selection_keeps_independent_geometry_and_stamps() {
    let (sketches, mut parameters) = separate_readers();
    let initial = sketches.solve_pending(&parameters, SolveOptions::default(), || false);
    assert_eq!(attempted(&initial), [1, 2]);
    let unchanged = initial
        .candidate
        .solve_pending(&parameters, SolveOptions::default(), || false);
    assert!(attempted(&unchanged).is_empty());
    assert!(initial.candidate.same_evaluated_state(&unchanged.candidate));
    parameters.set_expression(1, "6 cm").unwrap();
    let changed = initial
        .candidate
        .solve_pending(&parameters, SolveOptions::default(), || false);
    assert_eq!(attempted(&changed), [1]);
    assert!((height(&changed.candidate, 1, 3) - 6.0).abs() < 1e-8);
    assert_eq!(
        height(&changed.candidate, 2, 13).to_bits(),
        height(&initial.candidate, 2, 13).to_bits()
    );
}
#[test]
fn equal_model_values_do_not_recompute_verified_readers() {
    let (sketches, mut parameters) = separate_readers();
    let initial = sketches
        .solve(&parameters, SolveOptions::default(), || false)
        .candidate;
    parameters.set_expression(1, "50 mm").unwrap();
    let unchanged = initial.solve_pending(&parameters, Default::default(), || false);
    assert!(attempted(&unchanged).is_empty());
}
#[test]
fn current_failed_readers_skip_retries_until_their_inputs_recover() {
    let (initial, mut parameters) = verified_readers();
    parameters.set_expression(1, "missing + 1 cm").unwrap();
    let failed = initial.solve_pending(&parameters, SolveOptions::default(), || false);
    assert_eq!(attempted(&failed), [1]);
    assert!(failed.report.reports[0].result.is_err());
    let repeated = failed
        .candidate
        .solve_pending(&parameters, Default::default(), || false);
    assert!(attempted(&repeated).is_empty());
    parameters
        .insert_expression(3, "missing", USER_PARAM, "4 cm")
        .unwrap();
    let recovered = failed
        .candidate
        .solve_pending(&parameters, SolveOptions::default(), || false);
    assert_eq!(attempted(&recovered), [1]);
    assert!(recovered.report.reports[0].result.is_ok());
}
#[test]
fn pending_cancellation_keeps_geometry_and_evaluation_state_for_every_reader() {
    let (sketches, mut parameters) = separate_readers();
    let original = sketches
        .solve(&parameters, SolveOptions::default(), || false)
        .candidate;
    parameters.set_expression(1, "8 cm").unwrap();
    for stop_at in [1, 2] {
        let mut calls = 0;
        let cancelled = original.solve_pending(&parameters, SolveOptions::default(), || {
            calls += 1;
            calls == stop_at
        });
        assert!(cancelled.report.cancelled);
        assert!(cancelled.candidate.same_evaluated_state(&original));
    }
}
#[test]
fn evaluation_comparison_separates_health_updates_from_authored_changes() {
    let (sketches, parameters) = separate_readers();
    let solved = sketches
        .solve(&parameters, SolveOptions::default(), || false)
        .candidate;
    let restored = DocumentSketches::from_recipe(&solved.recipe()).unwrap();
    assert!(solved.same_recipe(&restored));
    assert!(!solved.same_evaluated_state(&restored));
    assert!(solved.same_evaluated_state(&solved.clone()));
    assert!(!solved.same_evaluated_state(&sketches));
}

#[test]
fn local_failed_reader_does_not_block_an_independent_changed_reader() {
    let (initial, mut parameters) = verified_readers();
    parameters.set_expression(1, "missing + 1 cm").unwrap();
    parameters.set_expression(2, "9 cm").unwrap();
    let changed = initial.solve_pending(&parameters, Default::default(), || false);
    assert_eq!(attempted(&changed), [1, 2]);
    assert!(changed.report.reports[0].result.is_err());
    assert_eq!(
        solved_report(&changed, 1).numerical().termination,
        SolveTermination::Converged
    );
    assert!((height(&changed.candidate, 2, 13) - 9.0).abs() < 1e-8);
    assert_eq!(
        height(&changed.candidate, 1, 3).to_bits(),
        height(&initial, 1, 3).to_bits()
    );
}

fn verified_readers() -> (DocumentSketches, ParameterCollection<u64>) {
    let (sketches, parameters) = separate_readers();
    let verified = sketches
        .solve(&parameters, Default::default(), || false)
        .candidate;
    (verified, parameters)
}
