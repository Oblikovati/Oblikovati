// SPDX-License-Identifier: GPL-2.0-only

use super::{
    FeatureEvaluationStatus, FeatureHealthRecord, FeatureReplayController, FeatureReplayError,
    FeatureReplayStep,
};
use crate::feature::FeatureId;

impl<Definition, Failure> FeatureReplayController<'_, Definition, Failure> {
    /// Classify the next entry once; example: `while let Some(step) = controller.advance()? { /* execute or pass through */ }`.
    pub fn advance(&mut self) -> Result<Option<FeatureReplayStep>, FeatureReplayError> {
        if let Some(feature) = self.pending {
            return Err(FeatureReplayError::Pending(feature));
        }
        if self.is_complete() {
            return Ok(None);
        }
        let action = self.action_for(&self.sequence.features()[self.next])?;
        self.accept_action(&action);
        Ok(Some(action))
    }

    /// Record an operation after kernel postcondition validation; example: `controller.record_result(id, validated.map(|_| ()))?`.
    pub fn record_result(
        &mut self,
        feature: FeatureId,
        result: Result<(), Failure>,
    ) -> Result<(), FeatureReplayError> {
        if self.pending != Some(feature) {
            return Err(FeatureReplayError::ResultIdentity {
                feature,
                expected: self.pending,
            });
        }
        let status = match result {
            Ok(()) => FeatureEvaluationStatus::Healthy,
            Err(failure) => FeatureEvaluationStatus::Failed(failure),
        };
        self.register(FeatureHealthRecord { feature, status });
        self.pending = None;
        Ok(())
    }

    fn accept_action(&mut self, action: &FeatureReplayStep) {
        match action {
            FeatureReplayStep::ConditionFailed { feature, error } => {
                self.record_condition_failure(*feature, error.clone())
            }
            FeatureReplayStep::Evaluate(feature) => self.pending = Some(*feature),
            FeatureReplayStep::Suppressed(feature) => self.register(FeatureHealthRecord {
                feature: *feature,
                status: FeatureEvaluationStatus::Suppressed,
            }),
            FeatureReplayStep::Blocked { feature, drivers } => self.register(FeatureHealthRecord {
                feature: *feature,
                status: FeatureEvaluationStatus::Blocked {
                    drivers: drivers.clone(),
                },
            }),
        }
    }

    fn record_condition_failure(
        &mut self,
        feature: FeatureId,
        error: crate::feature::FeatureSuppressionError,
    ) {
        let status = FeatureEvaluationStatus::ConditionFailed(error);
        self.register(FeatureHealthRecord { feature, status });
    }
}
