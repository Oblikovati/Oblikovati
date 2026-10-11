// SPDX-License-Identifier: GPL-2.0-only
use super::{WorkPlaneFeatureRecipe, WorkPlaneProgramRecipe, WorkPlaneProgramRecipeError};

pub(super) fn program_recipe_version(features: &[WorkPlaneFeatureRecipe]) -> u32 {
    if features.iter().any(|feature| {
        matches!(
            feature.definition.definition,
            super::super::WorkPlaneDefinitionRecipe::Axis(_)
        )
    }) {
        return 3;
    }
    if features.iter().any(|feature| feature.construction) {
        return 2;
    }
    1
}
pub(super) fn validate_program_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if !matches!(recipe.version, 1..=3) {
        return Err(WorkPlaneProgramRecipeError::Version(recipe.version));
    }
    validate_axis_version(recipe)?;
    if recipe.version == 1
        && let Some(feature) = recipe.features.iter().find(|feature| feature.construction)
    {
        return Err(WorkPlaneProgramRecipeError::LegacyConstruction(feature.id));
    }
    Ok(())
}

fn validate_axis_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if recipe.version < 3
        && let Some(feature) = recipe.features.iter().find(|feature| {
            matches!(
                feature.definition.definition,
                super::super::WorkPlaneDefinitionRecipe::Axis(_)
            )
        })
    {
        return Err(WorkPlaneProgramRecipeError::LegacyAxis(feature.id));
    }
    Ok(())
}
