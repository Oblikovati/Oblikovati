// SPDX-License-Identifier: GPL-2.0-only
//! Conditional suppression from Go `model/feature/suppression.go`, aa8297d1.

mod error;
mod recipe;
use super::{
    AuthoredFeature, FeatureId, FeatureReadKey, FeatureReplayRange, FeatureSequence,
    FeatureSequenceError,
};
use crate::{parameter::ParameterCollection, quantity::Quantity};
pub use error::FeatureSuppressionError;
pub use recipe::FeatureSuppressionRecipe;

/// Internal comparison semantics, independent of public numeric API constants.
#[derive(Clone, Copy, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum FeatureSuppressionComparison {
    Equal,
    NotEqual,
    LessThan,
    GreaterThan,
    LessOrEqual,
    GreaterOrEqual,
}

/// Stable parameter identity and finite numeric threshold in document working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct FeatureSuppressionCondition {
    parameter: u64,
    comparison: FeatureSuppressionComparison,
    threshold: Quantity,
}

impl FeatureSuppressionCondition {
    /// Create a dimensioned condition; example: `FeatureSuppressionCondition::new(1, FeatureSuppressionComparison::LessThan, Quantity::new(2.0, Dimension::Length))?`.
    pub fn new(
        parameter: u64,
        comparison: FeatureSuppressionComparison,
        threshold: Quantity,
    ) -> Result<Self, FeatureSuppressionError> {
        if !threshold.dimension.is_arithmetic() || !threshold.value.is_finite() {
            return Err(FeatureSuppressionError::Threshold {
                parameter,
                threshold,
            });
        }
        Ok(Self {
            parameter,
            comparison,
            threshold,
        })
    }

    /// Read the stable input ID; example: `condition.parameter()` survives rename.
    pub fn parameter(self) -> u64 {
        self.parameter
    }

    /// Read the comparison; example: `condition.comparison()` for the editor.
    pub fn comparison(self) -> FeatureSuppressionComparison {
        self.comparison
    }

    /// Read the authored working-unit threshold; example: `condition.threshold()`.
    pub fn threshold(self) -> Quantity {
        self.threshold
    }

    /// Read the selected model value without last-good fallback; example: `condition.evaluate(&parameters)?`.
    pub fn evaluate(
        self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<bool, FeatureSuppressionError> {
        let value = parameters
            .model_value(self.parameter)
            .map_err(FeatureSuppressionError::Parameter)?;
        if value.dimension != self.threshold.dimension {
            return Err(FeatureSuppressionError::Dimension {
                parameter: self.parameter,
                found: value.dimension,
                expected: self.threshold.dimension,
            });
        }
        Ok(self.compare_value(value.value))
    }

    fn compare_value(self, value: f64) -> bool {
        let threshold = self.threshold.value;
        match self.comparison {
            FeatureSuppressionComparison::Equal => value == threshold,
            FeatureSuppressionComparison::NotEqual => value != threshold,
            FeatureSuppressionComparison::LessThan => value < threshold,
            FeatureSuppressionComparison::GreaterThan => value > threshold,
            FeatureSuppressionComparison::LessOrEqual => value <= threshold,
            FeatureSuppressionComparison::GreaterOrEqual => value >= threshold,
        }
    }

    pub(super) fn same_recipe(self, other: Self) -> bool {
        self.parameter == other.parameter
            && self.comparison == other.comparison
            && self.threshold.dimension == other.threshold.dimension
            && self.threshold.value.to_bits() == other.threshold.value.to_bits()
    }
}

impl<Definition> AuthoredFeature<Definition> {
    pub(super) fn store_condition(&mut self, condition: Option<FeatureSuppressionCondition>) {
        self.condition = condition;
        self.reads = self.definition_reads.clone();
        if let Some(condition) = condition {
            self.reads
                .keys
                .insert(FeatureReadKey::Parameter(condition.parameter));
        }
    }
}

impl<Definition> FeatureSequence<Definition> {
    /// Set or clear a condition without losing definition reads; example: `sequence.set_suppression_condition(id, Some(condition))?`.
    pub fn set_suppression_condition(
        &mut self,
        id: FeatureId,
        condition: Option<FeatureSuppressionCondition>,
    ) -> Result<Option<FeatureReplayRange>, FeatureSequenceError> {
        let index = self.position(id)?;
        let unchanged = match (self.features[index].condition, condition) {
            (Some(first), Some(second)) => first.same_recipe(second),
            (None, None) => true,
            _ => false,
        };
        if unchanged {
            return Ok(None);
        }
        self.features[index].store_condition(condition);
        Ok(Some(self.replay_from(index)))
    }
}
