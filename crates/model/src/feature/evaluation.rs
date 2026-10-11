// SPDX-License-Identifier: GPL-2.0-only
//! Health protocol for ordered replay; bodies and result adoption have separate owners.

mod error;
mod prefix;
mod steps;

use super::FeatureSuppressionError;
use super::{AuthoredFeature, FeatureId, FeatureReplayRange, FeatureSequence};
use crate::parameter::ParameterCollection;
pub use error::{FeatureReplayError, FeatureReplayMismatch};
use std::collections::BTreeSet;

/// Derived replay health; a typed failure stays attached to its exact feature.
#[derive(Clone, Debug, PartialEq)]
pub enum FeatureEvaluationStatus<Failure> {
    Healthy,
    Suppressed,
    Failed(Failure),
    ConditionFailed(FeatureSuppressionError),
    Blocked { drivers: Vec<FeatureId> },
}

/// One ordered evaluation receipt, independent of authored definitions.
#[derive(Clone, Debug, PartialEq)]
pub struct FeatureHealthRecord<Failure> {
    pub feature: FeatureId,
    pub status: FeatureEvaluationStatus<Failure>,
}

/// Exactly one replay action; skipped entries must pass the running body state through.
#[derive(Clone, Debug, PartialEq)]
pub enum FeatureReplayStep {
    Evaluate(FeatureId),
    Suppressed(FeatureId),
    ConditionFailed {
        feature: FeatureId,
        error: FeatureSuppressionError,
    },
    Blocked {
        feature: FeatureId,
        drivers: Vec<FeatureId>,
    },
}

/// Sequential health owner for an immutable feature recipe and a validated prefix.
/// This protocol is not a generation receipt or a proof of kernel body isolation.
pub struct FeatureReplayController<'a, Definition, Failure> {
    sequence: &'a FeatureSequence<Definition>,
    range: FeatureReplayRange,
    next: usize,
    pending: Option<FeatureId>,
    unavailable: BTreeSet<FeatureId>,
    records: Vec<FeatureHealthRecord<Failure>>,
    parameters: Option<&'a ParameterCollection<u64>>,
}

impl<'a, Definition, Failure> FeatureReplayController<'a, Definition, Failure> {
    /// Seed health for the unchanged prefix; example: `FeatureReplayController::<_, KernelError>::new(&sequence, range, prefix)?`.
    pub fn new(
        sequence: &'a FeatureSequence<Definition>,
        range: FeatureReplayRange,
        prefix: Vec<FeatureHealthRecord<Failure>>,
    ) -> Result<Self, FeatureReplayError> {
        Self::with_inputs(sequence, range, prefix, None)
    }

    /// Supply immutable parameter model values for conditions; example: `FeatureReplayController::<_, KernelError>::new_with_parameters(&sequence, range, prefix, &parameters)?`.
    pub fn new_with_parameters(
        sequence: &'a FeatureSequence<Definition>,
        range: FeatureReplayRange,
        prefix: Vec<FeatureHealthRecord<Failure>>,
        parameters: &'a ParameterCollection<u64>,
    ) -> Result<Self, FeatureReplayError> {
        Self::with_inputs(sequence, range, prefix, Some(parameters))
    }

    fn with_inputs(
        sequence: &'a FeatureSequence<Definition>,
        range: FeatureReplayRange,
        prefix: Vec<FeatureHealthRecord<Failure>>,
        parameters: Option<&'a ParameterCollection<u64>>,
    ) -> Result<Self, FeatureReplayError> {
        Self::validate_range(sequence, range, prefix.len())?;
        let mut controller = Self::empty(sequence, range, parameters);
        for record in prefix {
            controller.accept_prefix(record)?;
        }
        Ok(controller)
    }

    fn empty(
        sequence: &'a FeatureSequence<Definition>,
        range: FeatureReplayRange,
        parameters: Option<&'a ParameterCollection<u64>>,
    ) -> Self {
        Self {
            sequence,
            range,
            next: 0,
            pending: None,
            unavailable: BTreeSet::new(),
            records: Vec::new(),
            parameters,
        }
    }

    /// Inspect ordered health without adopting any body output; example: `controller.records()` for diagnostics.
    pub fn records(&self) -> &[FeatureHealthRecord<Failure>] {
        &self.records
    }

    /// Inspect completion of all active entries; example: `controller.is_complete()` before owner adoption.
    pub fn is_complete(&self) -> bool {
        self.next == self.range.end && self.pending.is_none()
    }

    /// Consume a complete health report; example: `let records = controller.finish()?;`.
    pub fn finish(self) -> Result<Vec<FeatureHealthRecord<Failure>>, FeatureReplayError> {
        if !self.is_complete() {
            return Err(FeatureReplayError::Incomplete {
                next: self.next,
                end: self.range.end,
                pending: self.pending,
            });
        }
        Ok(self.records)
    }

    fn action_for(
        &self,
        feature: &AuthoredFeature<Definition>,
    ) -> Result<FeatureReplayStep, FeatureReplayError> {
        let drivers = self.unavailable_drivers(feature);
        // ADR-0004: suppression must not hide an unavailable dependency from later readers.
        if !drivers.is_empty() {
            return Ok(FeatureReplayStep::Blocked {
                feature: feature.id,
                drivers,
            });
        }
        if feature.suppressed {
            return Ok(FeatureReplayStep::Suppressed(feature.id));
        }
        self.conditional_action(feature)
    }

    fn unavailable_drivers(&self, feature: &AuthoredFeature<Definition>) -> Vec<FeatureId> {
        feature
            .reads
            .feature_drivers()
            .into_iter()
            .filter(|driver| self.unavailable.contains(driver))
            .collect()
    }

    fn conditional_action(
        &self,
        feature: &AuthoredFeature<Definition>,
    ) -> Result<FeatureReplayStep, FeatureReplayError> {
        let Some(condition) = feature.condition else {
            return Ok(FeatureReplayStep::Evaluate(feature.id));
        };
        let parameters = self
            .parameters
            .ok_or(FeatureReplayError::ParametersRequired(feature.id))?;
        let action = match condition.evaluate(parameters) {
            Ok(true) => FeatureReplayStep::Suppressed(feature.id),
            Ok(false) => FeatureReplayStep::Evaluate(feature.id),
            Err(error) => FeatureReplayStep::ConditionFailed {
                feature: feature.id,
                error,
            },
        };
        Ok(action)
    }

    fn register(&mut self, record: FeatureHealthRecord<Failure>) {
        if matches!(
            record.status,
            FeatureEvaluationStatus::Failed(_)
                | FeatureEvaluationStatus::ConditionFailed(_)
                | FeatureEvaluationStatus::Blocked { .. }
        ) {
            self.unavailable.insert(record.feature);
        }
        self.records.push(record);
        self.next += 1;
    }
}
