// SPDX-License-Identifier: GPL-2.0-only

use super::{
    FeatureEvaluationStatus, FeatureHealthRecord, FeatureReplayController, FeatureReplayError,
    FeatureReplayMismatch, FeatureReplayStep,
};
use crate::feature::{FeatureReplayRange, FeatureSequence};

impl<Definition, Failure> FeatureReplayController<'_, Definition, Failure> {
    pub(super) fn validate_range(
        sequence: &FeatureSequence<Definition>,
        range: FeatureReplayRange,
        length: usize,
    ) -> Result<(), FeatureReplayError> {
        let cutoff = sequence.effective_end();
        if range.start > range.end || range.end != cutoff {
            return Err(FeatureReplayError::Range { range, cutoff });
        }
        if length != range.start {
            return Err(FeatureReplayError::PrefixLength {
                length,
                start: range.start,
            });
        }
        Ok(())
    }

    pub(super) fn accept_prefix(
        &mut self,
        record: FeatureHealthRecord<Failure>,
    ) -> Result<(), FeatureReplayError> {
        let feature = &self.sequence.features()[self.next];
        if record.feature != feature.id {
            return Err(FeatureReplayError::PrefixIdentity {
                index: self.next,
                ids: FeatureReplayMismatch {
                    actual: record.feature,
                    expected: feature.id,
                },
            });
        }
        let expected = self.action_for(feature)?;
        Self::validate_prefix_status(&record, expected)?;
        self.register(record);
        Ok(())
    }

    fn validate_prefix_status(
        record: &FeatureHealthRecord<Failure>,
        expected: FeatureReplayStep,
    ) -> Result<(), FeatureReplayError> {
        if let (
            FeatureEvaluationStatus::ConditionFailed(actual),
            FeatureReplayStep::ConditionFailed { error, .. },
        ) = (&record.status, &expected)
        {
            if actual == error {
                return Ok(());
            }
            return Err(Self::prefix_status_error(record, &expected));
        }
        Self::validate_regular_prefix_status(record, expected)
    }

    fn validate_regular_prefix_status(
        record: &FeatureHealthRecord<Failure>,
        expected: FeatureReplayStep,
    ) -> Result<(), FeatureReplayError> {
        match (&record.status, &expected) {
            (
                FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Failed(_),
                FeatureReplayStep::Evaluate(_),
            ) => Ok(()),
            (FeatureEvaluationStatus::Suppressed, FeatureReplayStep::Suppressed(_)) => Ok(()),
            (
                FeatureEvaluationStatus::Blocked { drivers },
                FeatureReplayStep::Blocked {
                    drivers: expected, ..
                },
            ) => Self::validate_prefix_drivers(record.feature, drivers, expected),
            _ => Err(Self::prefix_status_error(record, &expected)),
        }
    }

    fn prefix_status_error(
        record: &FeatureHealthRecord<Failure>,
        expected: &FeatureReplayStep,
    ) -> FeatureReplayError {
        FeatureReplayError::PrefixStatus {
            feature: record.feature,
            statuses: FeatureReplayMismatch {
                actual: Self::status_label(&record.status),
                expected: Self::action_label(expected),
            },
        }
    }

    fn validate_prefix_drivers(
        feature: crate::feature::FeatureId,
        drivers: &[crate::feature::FeatureId],
        expected: &[crate::feature::FeatureId],
    ) -> Result<(), FeatureReplayError> {
        if drivers != expected {
            return Err(FeatureReplayError::PrefixDrivers {
                feature,
                drivers: FeatureReplayMismatch {
                    actual: drivers.to_vec(),
                    expected: expected.to_vec(),
                },
            });
        }
        Ok(())
    }

    fn status_label(status: &FeatureEvaluationStatus<Failure>) -> &'static str {
        match status {
            FeatureEvaluationStatus::Healthy => "healthy",
            FeatureEvaluationStatus::Suppressed => "suppressed",
            FeatureEvaluationStatus::Failed(_) => "failed",
            FeatureEvaluationStatus::ConditionFailed(_) => "failed suppression condition",
            FeatureEvaluationStatus::Blocked { .. } => "blocked",
        }
    }

    fn action_label(action: &FeatureReplayStep) -> &'static str {
        match action {
            FeatureReplayStep::Evaluate(_) => "healthy or failed",
            FeatureReplayStep::ConditionFailed { .. } => "the current suppression read failure",
            FeatureReplayStep::Suppressed(_) => "suppressed",
            FeatureReplayStep::Blocked { .. } => "blocked by unavailable drivers",
        }
    }
}
