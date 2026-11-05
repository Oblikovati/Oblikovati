// SPDX-License-Identifier: GPL-2.0-only

use super::*;
type Controller<'a> = FeatureReplayController<'a, u64, ()>;

fn controller<'a>(
    sequence: &'a FeatureSequence<u64>,
    parameters: &'a ParameterCollection<u64>,
    prefix: Vec<FeatureHealthRecord<()>>,
) -> Controller<'a> {
    let range = FeatureReplayRange {
        start: prefix.len(),
        end: sequence.effective_end(),
    };
    Controller::new_with_parameters(sequence, range, prefix, parameters).unwrap()
}

fn complete(mut controller: Controller<'_>) -> Vec<FeatureHealthRecord<()>> {
    while let Some(step) = controller.advance().unwrap() {
        if let FeatureReplayStep::Evaluate(feature) = step {
            controller.record_result(feature, Ok(())).unwrap();
        }
    }
    controller.finish().unwrap()
}

#[test]
fn changed_parameter_toggles_actual_replay_suppression() {
    let sequence = conditioned_sequence();
    let mut parameters = parameters(1.0, Dimension::Unitless);
    assert_eq!(
        complete(controller(&sequence, &parameters, vec![]))[0].status,
        FeatureEvaluationStatus::Suppressed
    );
    parameters
        .set_value(1, Quantity::new(3.0, Dimension::Unitless))
        .unwrap();
    assert_eq!(
        complete(controller(&sequence, &parameters, vec![]))[0].status,
        FeatureEvaluationStatus::Healthy
    );
}

#[test]
fn missing_condition_input_quarantines_readers() {
    let sequence = conditioned_sequence();
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let records = complete(controller(&sequence, &parameters, vec![]));
    assert!(matches!(
        records[0].status,
        FeatureEvaluationStatus::ConditionFailed(_)
    ));
    let blocked = FeatureEvaluationStatus::Blocked {
        drivers: vec![FeatureId(1)],
    };
    assert_eq!(records[1].status, blocked);
    assert_eq!(records[2].status, FeatureEvaluationStatus::Healthy);
    let reads = sequence.get(FeatureId(1)).unwrap().reads().keys();
    assert!(reads.contains(&FeatureReadKey::Parameter(1)));
}

#[test]
fn a_missing_condition_input_recovers_when_its_parameter_is_created() {
    let sequence = conditioned_sequence();
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let failed = complete(controller(&sequence, &parameters, vec![]));
    assert!(matches!(
        failed[0].status,
        FeatureEvaluationStatus::ConditionFailed(_)
    ));
    parameters
        .insert_value(
            1,
            "new",
            USER_PARAM,
            Quantity::new(3.0, Dimension::Unitless),
        )
        .unwrap();
    let recovered = complete(controller(&sequence, &parameters, vec![]));
    assert_eq!(recovered[0].status, FeatureEvaluationStatus::Healthy);
}

#[test]
fn failed_condition_prefix_is_checked_and_blocks_the_replay_tail() {
    let sequence = conditioned_sequence();
    let parameters = parameters(1.0, Dimension::Angle);
    let records = complete(controller(&sequence, &parameters, vec![]));
    let prefix = records.into_iter().take(1).collect();
    let report = complete(controller(&sequence, &parameters, prefix));
    assert_eq!(
        report[1].status,
        FeatureEvaluationStatus::Blocked {
            drivers: vec![FeatureId(1)]
        }
    );
}

#[test]
fn explicit_suppression_skips_the_condition_read_without_dropping_its_dependency() {
    let mut sequence = conditioned_sequence();
    sequence.set_suppressed(FeatureId(1), true).unwrap();
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let records = complete(controller(&sequence, &parameters, vec![]));
    assert_eq!(records[0].status, FeatureEvaluationStatus::Suppressed);
    assert!(
        sequence
            .replay_for_reads([FeatureReadKey::Parameter(1)])
            .is_some()
    );
}

#[test]
fn conditional_replay_without_parameter_inputs_refuses_to_advance() {
    let sequence = conditioned_sequence();
    let mut controller =
        Controller::new(&sequence, FeatureReplayRange { start: 0, end: 3 }, vec![]).unwrap();
    let error = FeatureReplayError::ParametersRequired(FeatureId(1));
    assert_eq!(controller.advance(), Err(error.clone()));
    assert!(controller.records().is_empty());
    assert!(
        error
            .to_string()
            .contains("FeatureId(1): expected immutable parameter model inputs")
    );
}

#[test]
fn a_stale_suppression_prefix_cannot_claim_a_live_condition_failure_is_healthy() {
    let sequence = conditioned_sequence();
    let parameters = parameters(1.0, Dimension::Angle);
    let prefix = vec![FeatureHealthRecord {
        feature: FeatureId(1),
        status: FeatureEvaluationStatus::Healthy,
    }];
    let range = FeatureReplayRange { start: 1, end: 3 };
    let error = Controller::new_with_parameters(&sequence, range, prefix, &parameters)
        .err()
        .unwrap();
    assert!(matches!(error, FeatureReplayError::PrefixStatus { .. }));
    assert!(
        error
            .to_string()
            .contains("expected the current suppression read failure")
    );
}

#[test]
fn a_prefix_cannot_retain_a_different_condition_failure() {
    let sequence = conditioned_sequence();
    let missing = ParameterCollection::new(UnitsOfMeasure::default());
    let old = complete(controller(&sequence, &missing, vec![]));
    let changed = parameters(1.0, Dimension::Angle);
    let prefix = old.into_iter().take(1).collect();
    let range = FeatureReplayRange { start: 1, end: 3 };
    assert!(Controller::new_with_parameters(&sequence, range, prefix, &changed).is_err());
}

#[test]
fn an_unconditional_prefix_cannot_claim_a_condition_failed() {
    let mut sequence = conditioned_sequence();
    let parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let old = complete(controller(&sequence, &parameters, vec![]));
    sequence
        .set_suppression_condition(FeatureId(1), None)
        .unwrap();
    let range = FeatureReplayRange { start: 1, end: 3 };
    let prefix = old.into_iter().take(1).collect();
    let error = Controller::new_with_parameters(&sequence, range, prefix, &parameters)
        .err()
        .unwrap();
    assert!(
        error
            .to_string()
            .contains("status failed suppression condition: expected healthy or failed")
    );
}
