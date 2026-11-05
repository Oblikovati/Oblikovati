// SPDX-License-Identifier: GPL-2.0-only
use super::{WorkPlaneFeatureRecipe, WorkPlaneProgramRecipe, WorkPlaneProgramRecipeError};

pub(super) fn program_recipe_version(features: &[WorkPlaneFeatureRecipe]) -> u32 {
    let definitions = features
        .iter()
        .map(|feature| feature.definition.definition.minimum_program_version())
        .max()
        .unwrap_or(1);
    definitions.max(metadata_program_version(features))
}
pub(super) fn validate_program_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if !matches!(recipe.version, 1..=17) {
        return Err(WorkPlaneProgramRecipeError::Version(recipe.version));
    }
    validate_definition_versions(recipe)?;
    validate_visibility_version(recipe)?;
    validate_ownership_version(recipe)?;
    validate_orientation_version(recipe)?;
    validate_display_version(recipe)?;
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

fn metadata_program_version(features: &[WorkPlaneFeatureRecipe]) -> u32 {
    let plane_version = plane_metadata_version(features);
    if plane_version > 1 {
        return plane_version;
    }
    if features
        .iter()
        .any(|feature| !feature.owned_scalars.is_empty())
    {
        return 7;
    }
    if features.iter().any(|feature| !feature.visible) {
        return 5;
    }
    if features.iter().any(|feature| feature.construction) {
        return 2;
    }
    1
}
fn validate_orientation_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if recipe.version < 8
        && let Some(feature) = recipe
            .features
            .iter()
            .find(|feature| feature.normal_flipped)
    {
        return Err(WorkPlaneProgramRecipeError::LegacyDefinition {
            feature: feature.id,
            minimum: 8,
        });
    }
    Ok(())
}
fn validate_visibility_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if recipe.version < 5
        && let Some(feature) = recipe.features.iter().find(|feature| !feature.visible)
    {
        return Err(WorkPlaneProgramRecipeError::LegacyVisibility(feature.id));
    }
    Ok(())
}

fn validate_ownership_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if recipe.version < 7
        && let Some(feature) = recipe
            .features
            .iter()
            .find(|feature| !feature.owned_scalars.is_empty())
    {
        return Err(WorkPlaneProgramRecipeError::LegacyDefinition {
            feature: feature.id,
            minimum: 7,
        });
    }
    Ok(())
}

fn validate_display_version(
    recipe: &WorkPlaneProgramRecipe,
) -> Result<(), WorkPlaneProgramRecipeError> {
    if recipe.version < 9
        && let Some(feature) = recipe
            .features
            .iter()
            .find(|feature| !feature.plane_display.is_default())
    {
        return Err(WorkPlaneProgramRecipeError::LegacyDefinition {
            feature: feature.id,
            minimum: 9,
        });
    }
    Ok(())
}

fn plane_metadata_version(features: &[WorkPlaneFeatureRecipe]) -> u32 {
    if features
        .iter()
        .any(|feature| !feature.plane_display.is_default())
    {
        return 9;
    }
    if features.iter().any(|feature| feature.normal_flipped) {
        return 8;
    }
    1
}
