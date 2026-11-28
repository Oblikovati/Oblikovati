// SPDX-License-Identifier: GPL-2.0-only
//! Typed Surface Trim creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        surface_trim::{
            SurfaceTrimDefinition, SurfaceTrimRecipe, SurfaceTrimSide, SurfaceTrimTool,
        },
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Surface Trim feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SurfaceTrimFeatureInputs {
    pub tool: SurfaceTrimTool,
    pub target_body: Vec<u8>,
    pub remove_side: SurfaceTrimSide,
}

/// A validated Surface Trim recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedSurfaceTrim {
    name: String,
    definition: SurfaceTrimDefinition,
}

impl PreparedSurfaceTrim {
    /// Inspect the validated Surface Trim definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &SurfaceTrimDefinition {
        &self.definition
    }

    /// Add the prepared Surface Trim recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_surface_trim(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Surface Trim definition from desktop/automation inputs.
/// Example: `build_surface_trim_definition(&inputs)?`.
pub fn build_surface_trim_definition(
    inputs: &SurfaceTrimFeatureInputs,
) -> Result<SurfaceTrimDefinition, Error> {
    let recipe = SurfaceTrimRecipe {
        version: 1,
        tool: inputs.tool.clone(),
        target_body: inputs.target_body.clone(),
        remove_side: inputs.remove_side,
    };
    SurfaceTrimDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Surface Trim feature (`Trim1`, `Trim2`, ...) without mutating the document.
/// Example: `prepare_surface_trim_definition(snapshot, &inputs)?`.
pub fn prepare_surface_trim_definition(
    snapshot: &DocumentSnapshot,
    inputs: &SurfaceTrimFeatureInputs,
) -> Result<PreparedSurfaceTrim, Error> {
    let definition = build_surface_trim_definition(inputs)?;
    Ok(PreparedSurfaceTrim {
        name: crate::fillet::next_name(snapshot, "Trim")?,
        definition,
    })
}

/// Replace an existing Surface Trim feature's definition inside a transaction/preview candidate.
/// Example: `replace_surface_trim(candidate, id, definition)?`.
pub fn replace_surface_trim(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SurfaceTrimDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_surface_trim(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
