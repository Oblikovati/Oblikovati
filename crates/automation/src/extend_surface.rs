// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Surface Extend creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        extend_surface::{
            ExtendSurfaceDefinition, ExtendSurfaceExtent, ExtendSurfaceMethod, ExtendSurfaceRecipe,
        },
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Surface Extend feature.
#[derive(Clone, Debug, PartialEq)]
pub struct ExtendSurfaceInputs {
    pub target_body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub edge_chain: bool,
    pub extent: ExtendSurfaceExtent,
    pub method: ExtendSurfaceMethod,
}

/// A validated Extend Surface recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedExtendSurface {
    name: String,
    definition: ExtendSurfaceDefinition,
}

impl PreparedExtendSurface {
    /// Inspect the generated feature name (`Extend1`, `Extend2`, ...).
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Inspect the validated Extend Surface definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &ExtendSurfaceDefinition {
        &self.definition
    }

    /// Add the prepared Extend Surface recipe to a private candidate or transaction.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_extend_surface(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Extend Surface definition from desktop/automation inputs.
/// Example: `build_extend_surface_definition(&inputs)?`.
pub fn build_extend_surface_definition(
    inputs: &ExtendSurfaceInputs,
) -> Result<ExtendSurfaceDefinition, Error> {
    let recipe = ExtendSurfaceRecipe {
        version: 1,
        target_body: inputs.target_body.clone(),
        edges: inputs.edges.clone(),
        edge_chain: inputs.edge_chain,
        extent: inputs.extent.clone(),
        method: inputs.method,
    };
    ExtendSurfaceDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Extend Surface feature (`Extend1`) without mutating the document.
/// Example: `prepare_extend_surface_definition(snapshot, &inputs)?`.
pub fn prepare_extend_surface_definition(
    snapshot: &DocumentSnapshot,
    inputs: &ExtendSurfaceInputs,
) -> Result<PreparedExtendSurface, Error> {
    let definition = build_extend_surface_definition(inputs)?;
    Ok(PreparedExtendSurface {
        name: crate::fillet::next_name(snapshot, "Extend")?,
        definition,
    })
}

/// Replace an existing Extend Surface feature's definition inside a transaction/preview candidate.
/// Example: `replace_extend_surface(candidate, id, definition)?`.
pub fn replace_extend_surface(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: ExtendSurfaceDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_extend_surface(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
