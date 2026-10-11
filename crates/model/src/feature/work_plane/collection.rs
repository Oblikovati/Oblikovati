// SPDX-License-Identifier: GPL-2.0-only

mod comparison;
mod error;
use super::{WorkPlaneDefinition, WorkPlaneRecipe};
use crate::feature::{
    AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureSequence, FeatureSuppressionCondition,
    FeatureSuppressionRecipe,
};
pub use error::WorkPlaneProgramRecipeError;

/// One authored feature; evaluated frames and health are not recipe contents.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneFeatureRecipe {
    pub id: FeatureId,
    pub name: String,
    pub definition: WorkPlaneRecipe,
    pub suppressed: bool,
    pub condition: Option<FeatureSuppressionRecipe>,
    pub definition_reads: FeatureReadFootprint,
}

/// Ordered collection schema; permanent feature reservations belong to the document owner.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneProgramRecipe {
    pub version: u32,
    pub features: Vec<WorkPlaneFeatureRecipe>,
    pub end_of_part: Option<usize>,
}

impl AuthoredFeature<WorkPlaneDefinition> {
    fn work_plane_record(&self) -> Result<WorkPlaneFeatureRecipe, WorkPlaneProgramRecipeError> {
        let definition = self.definition.recipe();
        WorkPlaneDefinition::from_recipe(definition).map_err(WorkPlaneProgramRecipeError::Plane)?;
        Ok(WorkPlaneFeatureRecipe {
            id: self.id,
            name: self.name.clone(),
            definition,
            suppressed: self.suppressed,
            condition: self.condition.map(FeatureSuppressionCondition::recipe),
            definition_reads: self.definition_reads.clone(),
        })
    }

    fn from_work_plane_record(
        recipe: &WorkPlaneFeatureRecipe,
    ) -> Result<Self, WorkPlaneProgramRecipeError> {
        let definition = WorkPlaneDefinition::from_recipe(recipe.definition)
            .map_err(WorkPlaneProgramRecipeError::Plane)?;
        let mut feature = Self::new(
            recipe.id,
            &recipe.name,
            definition,
            recipe.definition_reads.clone(),
        );
        feature.suppressed = recipe.suppressed;
        let condition = restore_condition(recipe.condition.as_ref())?;
        feature.store_condition(condition);
        Ok(feature)
    }
}

fn restore_condition(
    recipe: Option<&FeatureSuppressionRecipe>,
) -> Result<Option<FeatureSuppressionCondition>, WorkPlaneProgramRecipeError> {
    recipe
        .map(FeatureSuppressionCondition::from_recipe)
        .transpose()
        .map_err(WorkPlaneProgramRecipeError::Condition)
}

impl FeatureSequence<WorkPlaneDefinition> {
    /// Save validated authored order and metadata; example: `serde_json::to_string(&sequence.work_plane_recipe()?)?`.
    pub fn work_plane_recipe(&self) -> Result<WorkPlaneProgramRecipe, WorkPlaneProgramRecipeError> {
        self.validate_work_plane_reads()
            .map_err(WorkPlaneProgramRecipeError::Reads)?;
        let features = self
            .features
            .iter()
            .map(AuthoredFeature::work_plane_record)
            .collect::<Result<Vec<_>, _>>()?;
        Ok(WorkPlaneProgramRecipe {
            version: 1,
            features,
            end_of_part: self.end_of_part,
        })
    }

    /// Build a fresh owner and validate all dependencies; example: `FeatureSequence::from_work_plane_recipe(&recipe)?`.
    pub fn from_work_plane_recipe(
        recipe: &WorkPlaneProgramRecipe,
    ) -> Result<Self, WorkPlaneProgramRecipeError> {
        if recipe.version != 1 {
            return Err(WorkPlaneProgramRecipeError::Version(recipe.version));
        }
        let mut sequence = Self::default();
        for record in &recipe.features {
            sequence
                .append(AuthoredFeature::from_work_plane_record(record)?)
                .map_err(WorkPlaneProgramRecipeError::Sequence)?;
        }
        sequence
            .validate_work_plane_reads()
            .map_err(WorkPlaneProgramRecipeError::Reads)?;
        // A deletion can leave the authored cutoff past the current tail; effective_end clamps it (#3724).
        sequence.end_of_part = recipe.end_of_part;
        Ok(sequence)
    }
}
