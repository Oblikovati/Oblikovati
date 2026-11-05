// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::{height, parameters};
use super::*;
use oblikovati_model::document_sketch::{DocumentSketchSolveOutcome, PlacedSketchHealth};

fn selected_result(
    original: &DocumentSketches,
    identity: u64,
    options: SolveOptions,
) -> DocumentSketchSolveOutcome {
    original
        .solve_one(DocumentSketchId(identity), &parameters(), options, || false)
        .unwrap()
}
fn assert_untouched(original: &DocumentSketches, candidate: &DocumentSketches, identity: u64) {
    let identity = DocumentSketchId(identity);
    assert_eq!(
        candidate.get(identity).unwrap(),
        original.get(identity).unwrap()
    );
}
fn assert_verified(candidate: &DocumentSketches, identity: u64) {
    let placed = candidate.get(DocumentSketchId(identity)).unwrap();
    assert!(matches!(
        placed.health(&parameters()),
        PlacedSketchHealth::Current(_)
    ));
}
#[test]
fn selected_solve_updates_only_the_requested_geometry_and_verification() {
    let original = document();
    let outcome = selected_result(&original, 2, Default::default());
    assert_eq!(outcome.report.reports.len(), 1);
    assert_eq!(outcome.report.reports[0].sketch, DocumentSketchId(2));
    assert!((height(&outcome.candidate, 2, 13) - 5.0).abs() < 1e-8);
    assert_untouched(&original, &outcome.candidate, 1);
    assert_verified(&outcome.candidate, 2);
}
#[test]
fn unknown_identity_is_rejected_before_cancellation_or_numerical_work() {
    let original = document();
    let mut polls = 0;
    let identity = DocumentSketchId(99);
    let result = original.solve_one(identity, &parameters(), Default::default(), || {
        polls += 1;
        false
    });
    assert!(result.unwrap_err().to_string().contains("99"));
    assert_eq!(polls, 0);
}
fn cancelled_selected(original: &DocumentSketches, stop_at: usize) -> DocumentSketchSolveOutcome {
    let identity = DocumentSketchId(2);
    let mut polls = 0;
    original
        .solve_one(identity, &parameters(), Default::default(), || {
            polls += 1;
            polls == stop_at
        })
        .unwrap()
}
#[test]
fn selected_cancellation_retains_all_geometry_and_health() {
    let original = document();
    for stop_at in [1, 3] {
        let outcome = cancelled_selected(&original, stop_at);
        assert!(outcome.report.cancelled);
        assert!(original.same_evaluated_state(&outcome.candidate));
    }
}
#[test]
fn selected_solve_retries_a_current_failure_with_the_new_policy() {
    let original = document();
    let invalid = SolveOptions {
        max_trial_steps: 0,
        ..Default::default()
    };
    let failed = selected_result(&original, 1, invalid);
    assert!(failed.report.reports[0].result.is_err());
    assert!(original.same_recipe(&failed.candidate));
    let retried = selected_result(&failed.candidate, 1, Default::default());
    assert_eq!(retried.report.reports.len(), 1);
    let numerical = retried.report.reports[0]
        .result
        .as_ref()
        .unwrap()
        .numerical();
    assert_eq!(numerical.termination, SolveTermination::Converged);
    assert!((height(&retried.candidate, 1, 3) - 5.0).abs() < 1e-8);
    assert_untouched(&original, &retried.candidate, 2);
}
