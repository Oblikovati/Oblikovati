// SPDX-License-Identifier: GPL-2.0-only

mod comparison;
mod error;
mod version;
use super::{WorkPlaneDefinition, WorkPlaneRecipe};
use crate::feature::{
    AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureSequence, FeatureSuppressionCondition,
    FeatureSuppressionRecipe,
};
pub use error::WorkPlaneProgramRecipeError;
use version::{program_recipe_version, validate_program_version};

/// One authored feature; evaluated frames and health are not recipe contents.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneFeatureRecipe {
    pub id: FeatureId,
    pub name: String,
    pub definition: WorkPlaneRecipe,
    pub suppressed: bool,
    // Older strict readers must still accept ordinary version-1 records (#3734).
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub construction: bool,
    #[serde(
        default = "default_datum_visibility",
        skip_serializing_if = "is_default_datum_visibility"
    )]
    pub visible: bool,
    pub condition: Option<FeatureSuppressionRecipe>,
    pub definition_reads: FeatureReadFootprint,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub owned_scalars: Vec<crate::feature::FeatureScalarBinding>,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub normal_flipped: bool,
    #[serde(
        default,
        skip_serializing_if = "crate::feature::work_geometry::WorkPlaneDisplaySettings::is_default"
    )]
    pub plane_display: crate::feature::work_geometry::WorkPlaneDisplaySettings,
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
        self.validate_plane_metadata()
            .map_err(WorkPlaneProgramRecipeError::Plane)?;
        let definition = self.definition.recipe();
        WorkPlaneDefinition::from_recipe(definition).map_err(WorkPlaneProgramRecipeError::Plane)?;
        Ok(WorkPlaneFeatureRecipe {
            id: self.id,
            name: self.name.clone(),
            definition,
            suppressed: self.suppressed,
            construction: self.construction_datum,
            visible: self.datum_visible,
            condition: self.condition.map(FeatureSuppressionCondition::recipe),
            definition_reads: self.definition_reads.clone(),
            owned_scalars: self.owned_scalars.clone(),
            normal_flipped: self.plane_normal_flipped,
            plane_display: self.plane_display.clone(),
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
        feature.restore_datum_metadata(recipe)?;
        let condition = restore_condition(recipe.condition.as_ref())?;
        feature.store_condition(condition);
        Ok(feature)
    }
    fn restore_datum_metadata(
        &mut self,
        recipe: &WorkPlaneFeatureRecipe,
    ) -> Result<(), WorkPlaneProgramRecipeError> {
        self.suppressed = recipe.suppressed;
        self.construction_datum = recipe.construction;
        self.datum_visible = recipe.visible;
        self.owned_scalars = recipe.owned_scalars.clone();
        self.plane_normal_flipped = recipe.normal_flipped;
        self.plane_display = recipe.plane_display.clone();
        self.validate_plane_metadata()
            .map_err(WorkPlaneProgramRecipeError::Plane)
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
        let features: Vec<WorkPlaneFeatureRecipe> = self
            .features
            .iter()
            .map(AuthoredFeature::work_plane_record)
            .collect::<Result<_, _>>()?;
        let version = program_recipe_version(&features);
        Ok(WorkPlaneProgramRecipe {
            version,
            features,
            end_of_part: self.end_of_part,
        })
    }

    /// Build a fresh owner and validate all dependencies; example: `FeatureSequence::from_work_plane_recipe(&recipe)?`.
    pub fn from_work_plane_recipe(
        recipe: &WorkPlaneProgramRecipe,
    ) -> Result<Self, WorkPlaneProgramRecipeError> {
        validate_program_version(recipe)?;
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

fn default_datum_visibility() -> bool {
    true
}
fn is_default_datum_visibility(visible: &bool) -> bool {
    *visible
}
