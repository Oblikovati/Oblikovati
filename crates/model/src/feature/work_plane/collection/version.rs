// SPDX-License-Identifier: GPL-2.0-only
use super::{WorkPlaneFeatureRecipe, WorkPlaneProgramRecipe, WorkPlaneProgramRecipeError};

pub(super) fn program_recipe_version(features: &[WorkPlaneFeatureRecipe]) -> u32 {
    let definitions = features
        .iter()
        .map(|feature| feature.definition.definition.minimum_program_version())
        .max()
        .unwrap_or(1);
    let construction = if features.iter().any(|feature| feature.construction) {
        2
    } else {
        1
    };
    definitions.max(construction)
}
pub(super) fn validate_program_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if !matches!(recipe.version, 1..=4) {
        return Err(WorkPlaneProgramRecipeError::Version(recipe.version));
    }
    validate_definition_versions(recipe)?;
    if recipe.version == 1
        && let Some(feature) = recipe.features.iter().find(|feature| feature.construction)
    {
        return Err(WorkPlaneProgramRecipeError::LegacyConstruction(feature.id));
    }
    Ok(())
}
fn validate_definition_versions(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    for feature in &recipe.features {
        let minimum = feature.definition.definition.minimum_program_version();
        if recipe.version < minimum {
            return Err(WorkPlaneProgramRecipeError::LegacyDefinition {
                feature: feature.id,
                minimum,
            });
        }
    }
    Ok(())
}
