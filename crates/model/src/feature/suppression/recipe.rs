// SPDX-License-Identifier: GPL-2.0-only

use super::{FeatureSuppressionComparison, FeatureSuppressionCondition, FeatureSuppressionError};
use crate::{
    quantity::Quantity,
    units::{category_name, parse_category},
};

/// Authored threshold in working units; versioning belongs to the enclosing feature program.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FeatureSuppressionRecipe {
    pub parameter: u64,
    pub comparison: FeatureSuppressionComparison,
    pub threshold: f64,
    pub category: String,
}

impl FeatureSuppressionCondition {
    /// Capture the stable input and canonical threshold category; example: `condition.recipe()`.
    pub fn recipe(self) -> FeatureSuppressionRecipe {
        FeatureSuppressionRecipe {
            parameter: self.parameter,
            comparison: self.comparison,
            threshold: self.threshold.value,
            category: category_name(self.threshold.dimension).into(),
        }
    }

    /// Restore through numeric threshold validation; example: `FeatureSuppressionCondition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &FeatureSuppressionRecipe) -> Result<Self, FeatureSuppressionError> {
        let dimension =
            parse_category(&recipe.category).ok_or_else(|| FeatureSuppressionError::Category {
                parameter: recipe.parameter,
                category: recipe.category.clone(),
            })?;
        let threshold = Quantity::new(recipe.threshold, dimension);
        Self::new(recipe.parameter, recipe.comparison, threshold)
    }
}
