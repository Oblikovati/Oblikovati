// SPDX-License-Identifier: GPL-2.0-only

use super::*;

fn suffix<'a>(
    sequence: &'a FeatureSequence<u64>,
    prefix: Vec<ReplayRecord>,
) -> Result<ReplayController<'a>, FeatureReplayError> {
    ReplayController::new(
        sequence,
        FeatureReplayRange {
            start: prefix.len(),
            end: sequence.effective_end(),
        },
        prefix,
    )
}

#[test]
fn failed_clean_prefix_blocks_readers_without_retrying_the_prefix() {
    let sequence = diamond();
    let prefix = vec![failed_record(1, 10)];
    let mut controller = suffix(&sequence, prefix).unwrap();
    assert_eq!(
        controller.advance().unwrap(),
        Some(FeatureReplayStep::Blocked {
            feature: FeatureId(2),
            drivers: vec![FeatureId(1)]
        })
    );
    let records = complete(controller);
    assert_eq!(records[4].status, FeatureEvaluationStatus::Healthy);
}

#[test]
fn blocked_clean_prefix_propagates_exact_immediate_causes_in_stable_order() {
    let sequence = diamond();
    let prefix = vec![
        failed_record(1, 10),
        blocked_record(2, &[1]),
        blocked_record(3, &[1]),
    ];
    let mut controller = suffix(&sequence, prefix).unwrap();
    assert_eq!(
        controller.advance().unwrap(),
        Some(FeatureReplayStep::Blocked {
            feature: FeatureId(4),
            drivers: vec![FeatureId(2), FeatureId(3)]
        })
    );
    assert_eq!(complete(controller).len(), 5);
}

#[test]
fn suppressed_clean_prefix_is_retained_without_execution() {
    let mut sequence = diamond();
    sequence.set_suppressed(FeatureId(1), true).unwrap();
    let prefix = vec![record(1, FeatureEvaluationStatus::Suppressed)];
    let mut controller = suffix(&sequence, prefix).unwrap();
    evaluate(&mut controller, 2, Ok(()));
    assert_eq!(
        complete(controller)[0].status,
        FeatureEvaluationStatus::Suppressed
    );
}

#[test]
fn blocked_suppressed_prefix_does_not_release_downstream_readers() {
    let mut sequence = diamond();
    sequence.set_suppressed(FeatureId(2), true).unwrap();
    let prefix = vec![
        failed_record(1, 10),
        blocked_record(2, &[1]),
        blocked_record(3, &[1]),
    ];
    let mut controller = suffix(&sequence, prefix).unwrap();
    let expected = FeatureReplayStep::Blocked {
        feature: FeatureId(4),
        drivers: vec![FeatureId(2), FeatureId(3)],
    };
    assert_eq!(controller.advance().unwrap(), Some(expected));
    assert_eq!(
        complete(controller)[4].status,
        FeatureEvaluationStatus::Healthy
    );
}

#[test]
fn a_complete_clean_prefix_can_finish_an_empty_replay_tail() {
    let sequence = diamond();
    let records = complete(full_replay(&sequence));
    let mut controller = suffix(&sequence, records).unwrap();
    assert!(controller.is_complete());
    assert_eq!(controller.advance().unwrap(), None);
    assert_eq!(controller.finish().unwrap().len(), 5);
}

#[test]
fn prefix_length_must_match_the_exact_ordered_prefix() {
    let sequence = diamond();
    let range = FeatureReplayRange { start: 1, end: 5 };
    let error = ReplayController::new(&sequence, range, vec![])
        .err()
        .unwrap();
    assert_eq!(
        error,
        FeatureReplayError::PrefixLength {
            length: 0,
            start: 1
        }
    );
    assert!(
        error
            .to_string()
            .contains("prefix health length 0: expected replay start 1")
    );
}

#[test]
fn prefix_identity_must_match_the_exact_ordered_prefix() {
    let sequence = diamond();
    let error = suffix(&sequence, vec![record(9, FeatureEvaluationStatus::Healthy)])
        .err()
        .unwrap();
    assert_eq!(
        error,
        FeatureReplayError::PrefixIdentity {
            index: 0,
            ids: FeatureReplayMismatch {
                actual: FeatureId(9),
                expected: FeatureId(1)
            }
        }
    );
    let text = error.to_string();
    assert!(text.contains("feature FeatureId(9): expected FeatureId(1)"));
}

fn invalid_prefix_status(
    sequence: &FeatureSequence<u64>,
    prefix: Vec<ReplayRecord>,
    observed: &'static str,
    expected: &'static str,
) {
    let feature = prefix.last().unwrap().feature;
    let error = suffix(sequence, prefix).err().unwrap();
    let expected_error = FeatureReplayError::PrefixStatus {
        feature,
        statuses: FeatureReplayMismatch {
            actual: observed,
            expected,
        },
    };
    assert_eq!(error, expected_error);
    let expected_text = format!("status {observed}: expected {expected}");
    assert!(error.to_string().contains(&expected_text));
}

#[test]
fn prefix_health_cannot_hide_unavailable_dependencies() {
    let sequence = diamond();
    for status in [
        FeatureEvaluationStatus::Healthy,
        FeatureEvaluationStatus::Failed(ReplayFault::LostEntity(5)),
    ] {
        let observed = if status == FeatureEvaluationStatus::Healthy {
            "healthy"
        } else {
            "failed"
        };
        invalid_prefix_status(
            &sequence,
            vec![failed_record(1, 8), record(2, status)],
            observed,
            "blocked by unavailable drivers",
        );
    }
}

#[test]
fn an_unsuppressed_prefix_cannot_claim_suppression() {
    let sequence = diamond();
    invalid_prefix_status(
        &sequence,
        vec![record(1, FeatureEvaluationStatus::Suppressed)],
        "suppressed",
        "healthy or failed",
    );
}

#[test]
fn healthy_prefix_cannot_claim_an_authored_suppression_was_evaluated() {
    let mut sequence = diamond();
    invalid_prefix_status(
        &sequence,
        vec![blocked_record(1, &[9])],
        "blocked",
        "healthy or failed",
    );
    sequence.set_suppressed(FeatureId(1), true).unwrap();
    invalid_prefix_status(
        &sequence,
        vec![record(1, FeatureEvaluationStatus::Healthy)],
        "healthy",
        "suppressed",
    );
}

#[test]
fn incorrect_blocked_causes_cannot_seed_a_clean_prefix() {
    let sequence = diamond();
    let prefix = vec![failed_record(1, 8), blocked_record(2, &[9])];
    let error = suffix(&sequence, prefix).err().unwrap();
    assert_eq!(
        error,
        FeatureReplayError::PrefixDrivers {
            feature: FeatureId(2),
            drivers: FeatureReplayMismatch {
                actual: vec![FeatureId(9)],
                expected: vec![FeatureId(1)]
            }
        }
    );
    assert!(
        error
            .to_string()
            .contains("blocked drivers [FeatureId(9)]: expected [FeatureId(1)]")
    );
}
