// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::feature::*;

mod feature_replay {
    use super::*;
    mod exhaustive;
    mod prefix;
    mod protocol;
}

#[derive(Debug, Eq, PartialEq)]
enum ReplayFault {
    LostEntity(u64),
}

type ReplayController<'a> = FeatureReplayController<'a, u64, ReplayFault>;
type ReplayRecord = FeatureHealthRecord<ReplayFault>;

fn authored(id: u64, drivers: &[u64]) -> AuthoredFeature<u64> {
    let keys = drivers
        .iter()
        .map(|driver| FeatureReadKey::Feature(FeatureId(*driver)));
    AuthoredFeature::new(
        FeatureId(id),
        "Feature",
        id,
        FeatureReadFootprint::new(keys, false),
    )
}

fn diamond() -> FeatureSequence<u64> {
    let mut sequence = FeatureSequence::default();
    for (id, drivers) in [
        (1, vec![]),
        (2, vec![1]),
        (3, vec![1]),
        (4, vec![2, 3]),
        (5, vec![]),
    ] {
        sequence.append(authored(id, &drivers)).unwrap();
    }
    sequence
}

fn full_replay(sequence: &FeatureSequence<u64>) -> ReplayController<'_> {
    ReplayController::new(
        sequence,
        FeatureReplayRange {
            start: 0,
            end: sequence.effective_end(),
        },
        vec![],
    )
    .unwrap()
}

fn record(feature: u64, status: FeatureEvaluationStatus<ReplayFault>) -> ReplayRecord {
    FeatureHealthRecord {
        feature: FeatureId(feature),
        status,
    }
}

fn blocked_record(feature: u64, drivers: &[u64]) -> ReplayRecord {
    let drivers = drivers.iter().copied().map(FeatureId).collect();
    record(feature, FeatureEvaluationStatus::Blocked { drivers })
}

fn failed_record(feature: u64, entity: u64) -> ReplayRecord {
    let failure = ReplayFault::LostEntity(entity);
    record(feature, FeatureEvaluationStatus::Failed(failure))
}

fn evaluate(controller: &mut ReplayController<'_>, feature: u64, result: Result<(), ReplayFault>) {
    assert_eq!(
        controller.advance().unwrap(),
        Some(FeatureReplayStep::Evaluate(FeatureId(feature)))
    );
    controller
        .record_result(FeatureId(feature), result)
        .unwrap();
}

fn complete(mut controller: ReplayController<'_>) -> Vec<ReplayRecord> {
    while let Some(step) = controller.advance().unwrap() {
        if let FeatureReplayStep::Evaluate(feature) = step {
            controller.record_result(feature, Ok(())).unwrap();
        }
    }
    assert!(controller.is_complete());
    controller.finish().unwrap()
}

#[test]
fn a_failed_feature_quarantines_a_diamond_and_preserves_an_independent_branch() {
    let sequence = diamond();
    let mut controller = full_replay(&sequence);
    evaluate(&mut controller, 1, Err(ReplayFault::LostEntity(42)));
    let records = complete(controller);
    assert_eq!(records[0], failed_record(1, 42));
    assert_eq!(records[1], blocked_record(2, &[1]));
    assert_eq!(records[2], blocked_record(3, &[1]));
    assert_eq!(records[3], blocked_record(4, &[2, 3]));
    assert_eq!(records[4], record(5, FeatureEvaluationStatus::Healthy));
}

#[test]
fn a_successful_new_replay_does_not_retain_old_quarantine() {
    let sequence = diamond();
    let mut failed = full_replay(&sequence);
    evaluate(&mut failed, 1, Err(ReplayFault::LostEntity(42)));
    assert!(matches!(
        complete(failed)[3].status,
        FeatureEvaluationStatus::Blocked { .. }
    ));
    let recovered = complete(full_replay(&sequence));
    assert!(
        recovered
            .iter()
            .all(|entry| entry.status == FeatureEvaluationStatus::Healthy)
    );
}

#[test]
fn suppression_skips_execution_and_preserves_a_healthy_dependency_path() {
    let mut sequence = diamond();
    sequence.set_suppressed(FeatureId(2), true).unwrap();
    let mut controller = full_replay(&sequence);
    evaluate(&mut controller, 1, Ok(()));
    assert_eq!(
        controller.advance().unwrap(),
        Some(FeatureReplayStep::Suppressed(FeatureId(2)))
    );
    let records = complete(controller);
    assert_eq!(records[1].status, FeatureEvaluationStatus::Suppressed);
    assert_eq!(records[3].status, FeatureEvaluationStatus::Healthy);
}

#[test]
fn suppression_cannot_mask_a_failed_dependency_from_its_readers() {
    let mut sequence = diamond();
    sequence.set_suppressed(FeatureId(2), true).unwrap();
    let mut controller = full_replay(&sequence);
    evaluate(&mut controller, 1, Err(ReplayFault::LostEntity(42)));
    let records = complete(controller);
    assert_eq!(
        records[1].status,
        FeatureEvaluationStatus::Blocked {
            drivers: vec![FeatureId(1)]
        }
    );
    assert_eq!(
        records[3].status,
        FeatureEvaluationStatus::Blocked {
            drivers: vec![FeatureId(2), FeatureId(3)]
        }
    );
    assert!(sequence.get(FeatureId(2)).unwrap().suppressed());
}

#[test]
fn replay_does_not_change_authored_definitions_or_execute_beyond_the_cutoff() {
    let mut sequence = diamond();
    sequence.set_end_of_part(Some(2)).unwrap();
    let records = complete(full_replay(&sequence));
    assert_eq!(records.len(), 2);
    assert_eq!(sequence.features().len(), 5);
    assert_eq!(*sequence.get(FeatureId(4)).unwrap().definition(), 4);
}
