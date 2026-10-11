// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn result_mismatch(feature: u64, pending: Option<u64>) -> FeatureReplayError {
    let expected = pending.map(FeatureId);
    FeatureReplayError::ResultIdentity {
        feature: FeatureId(feature),
        expected,
    }
}

#[test]
fn pending_evaluation_cannot_be_skipped_or_completed_with_a_different_id() {
    let sequence = diamond();
    let mut controller = full_replay(&sequence);
    assert_eq!(
        controller.advance().unwrap(),
        Some(FeatureReplayStep::Evaluate(FeatureId(1)))
    );
    assert_eq!(
        controller.advance(),
        Err(FeatureReplayError::Pending(FeatureId(1)))
    );
    assert_eq!(
        controller.record_result(FeatureId(5), Ok(())),
        Err(result_mismatch(5, Some(1)))
    );
    assert!(controller.records().is_empty());
    controller.record_result(FeatureId(1), Ok(())).unwrap();
    assert_eq!(complete(controller).len(), 5);
}

#[test]
fn an_unsolicited_or_duplicate_result_does_not_advance_replay() {
    let sequence = diamond();
    let mut controller = full_replay(&sequence);
    assert_eq!(
        controller.record_result(FeatureId(1), Ok(())),
        Err(result_mismatch(1, None))
    );
    evaluate(&mut controller, 1, Ok(()));
    assert!(
        controller
            .record_result(FeatureId(1), Err(ReplayFault::LostEntity(8)))
            .is_err()
    );
    assert_eq!(
        controller.records(),
        [record(1, FeatureEvaluationStatus::Healthy)]
    );
    assert_eq!(complete(controller).len(), 5);
}

#[test]
fn incomplete_replay_cannot_produce_a_completed_report() {
    let sequence = diamond();
    let controller = full_replay(&sequence);
    assert!(!controller.is_complete());
    assert_eq!(
        controller.finish(),
        Err(FeatureReplayError::Incomplete {
            next: 0,
            end: 5,
            pending: None
        })
    );
}

#[test]
fn pending_replay_cannot_produce_a_completed_report() {
    let sequence = diamond();
    let mut pending = full_replay(&sequence);
    pending.advance().unwrap();
    assert_eq!(
        pending.finish(),
        Err(FeatureReplayError::Incomplete {
            next: 0,
            end: 5,
            pending: Some(FeatureId(1))
        })
    );
}

#[test]
fn skipped_entries_do_not_accept_backend_results() {
    let mut sequence = diamond();
    sequence.set_suppressed(FeatureId(1), true).unwrap();
    let mut controller = full_replay(&sequence);
    assert_eq!(
        controller.advance().unwrap(),
        Some(FeatureReplayStep::Suppressed(FeatureId(1)))
    );
    assert!(controller.record_result(FeatureId(1), Ok(())).is_err());
    assert_eq!(
        controller.records()[0].status,
        FeatureEvaluationStatus::Suppressed
    );
    assert_eq!(complete(controller).len(), 5);
}

#[test]
fn a_zero_cutoff_finishes_without_an_evaluation_and_remains_finished() {
    let mut sequence = diamond();
    sequence.set_end_of_part(Some(0)).unwrap();
    let mut controller = full_replay(&sequence);
    assert!(controller.is_complete());
    assert_eq!(controller.advance().unwrap(), None);
    assert_eq!(controller.advance().unwrap(), None);
    assert!(controller.finish().unwrap().is_empty());
}

#[test]
fn range_checks_refuse_truncated_or_reversed_replay() {
    let sequence = diamond();
    for range in [
        FeatureReplayRange { start: 0, end: 4 },
        FeatureReplayRange { start: 6, end: 5 },
    ] {
        let error = ReplayController::new(&sequence, range, vec![])
            .err()
            .unwrap();
        assert_eq!(error, FeatureReplayError::Range { range, cutoff: 5 });
        assert!(
            error
                .to_string()
                .contains("expected start <= end and end = cutoff 5")
        );
    }
}

#[test]
fn protocol_errors_include_the_offending_identity_and_expected_state() {
    let pending = FeatureReplayError::Pending(FeatureId(7));
    assert!(
        pending
            .to_string()
            .contains("FeatureId(7): expected its validated result")
    );
}

#[test]
fn result_errors_name_the_offending_and_pending_identities() {
    let result = FeatureReplayError::ResultIdentity {
        feature: FeatureId(8),
        expected: Some(FeatureId(7)),
    };
    assert!(
        result
            .to_string()
            .contains("result feature FeatureId(8): expected pending feature Some(FeatureId(7))")
    );
}

#[test]
fn incomplete_errors_name_the_expected_cutoff() {
    let unfinished = FeatureReplayError::Incomplete {
        next: 1,
        end: 3,
        pending: None,
    };
    assert!(
        unfinished
            .to_string()
            .contains("replay next 1, pending None: expected completed cutoff 3")
    );
}
