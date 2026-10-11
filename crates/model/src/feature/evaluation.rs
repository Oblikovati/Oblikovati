// SPDX-License-Identifier: GPL-2.0-only
//! Health protocol for ordered replay; bodies and result adoption have separate owners.

mod error;
mod prefix;
mod steps;

use super::{AuthoredFeature, FeatureId, FeatureReplayRange, FeatureSequence};
pub use error::{FeatureReplayError, FeatureReplayMismatch};
use std::collections::BTreeSet;

/// Derived replay health; a typed failure stays attached to its exact feature.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum FeatureEvaluationStatus<Failure> {
    Healthy,
    Suppressed,
    Failed(Failure),
    Blocked { drivers: Vec<FeatureId> },
}

/// One ordered evaluation receipt, independent of authored definitions.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FeatureHealthRecord<Failure> {
    pub feature: FeatureId,
    pub status: FeatureEvaluationStatus<Failure>,
}

/// Exactly one replay action; skipped entries must pass the running body state through.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum FeatureReplayStep {
    Evaluate(FeatureId),
    Suppressed(FeatureId),
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
}

impl<'a, Definition, Failure> FeatureReplayController<'a, Definition, Failure> {
    /// Seed health for the unchanged prefix; example: `FeatureReplayController::<_, KernelError>::new(&sequence, range, prefix)?`.
    pub fn new(
        sequence: &'a FeatureSequence<Definition>,
        range: FeatureReplayRange,
        prefix: Vec<FeatureHealthRecord<Failure>>,
    ) -> Result<Self, FeatureReplayError> {
        Self::validate_range(sequence, range, prefix.len())?;
        let mut controller = Self::empty(sequence, range);
        for record in prefix {
            controller.accept_prefix(record)?;
        }
        Ok(controller)
    }

    fn empty(sequence: &'a FeatureSequence<Definition>, range: FeatureReplayRange) -> Self {
        Self {
            sequence,
            range,
            next: 0,
            pending: None,
            unavailable: BTreeSet::new(),
            records: Vec::new(),
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

    fn action_for(&self, feature: &AuthoredFeature<Definition>) -> FeatureReplayStep {
        let drivers = feature
            .reads
            .feature_drivers()
            .into_iter()
            .filter(|driver| self.unavailable.contains(driver))
            .collect::<Vec<_>>();
        // ADR-0004: suppression must not hide an unavailable dependency from later readers.
        if !drivers.is_empty() {
            return FeatureReplayStep::Blocked {
                feature: feature.id,
                drivers,
            };
        }
        if feature.suppressed {
            return FeatureReplayStep::Suppressed(feature.id);
        }
        FeatureReplayStep::Evaluate(feature.id)
    }

    fn register(&mut self, record: FeatureHealthRecord<Failure>) {
        if matches!(
            record.status,
            FeatureEvaluationStatus::Failed(_) | FeatureEvaluationStatus::Blocked { .. }
        ) {
            self.unavailable.insert(record.feature);
        }
        self.records.push(record);
        self.next += 1;
    }
}
