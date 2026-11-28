// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Surface Ruled Surface creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        ruled_surface::{
            RuledSurfaceDefinition, RuledSurfaceRecipe, RuledSurfaceType, RuledSurfaceVector,
        },
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Surface Ruled Surface feature.
#[derive(Clone, Debug, PartialEq)]
pub struct RuledSurfaceInputs {
    pub target_body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub ruled_type: RuledSurfaceType,
    /// Ruled extension distance in model units (cm).
    pub distance: f64,
    /// Tilt/draft angle in degrees within [-180, 180].
    pub angle_deg: f64,
    pub flip: bool,
    pub automatic_edge_chain: bool,
    pub vector: Option<RuledSurfaceVector>,
}

/// A validated Ruled Surface recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedRuledSurface {
    name: String,
    definition: RuledSurfaceDefinition,
}

impl PreparedRuledSurface {
    /// Inspect the generated feature name (`Ruled Surface1`, `Ruled Surface2`, ...).
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Inspect the validated Ruled Surface definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &RuledSurfaceDefinition {
        &self.definition
    }

    /// Add the prepared Ruled Surface recipe to a private candidate.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_ruled_surface(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Ruled Surface definition from desktop/automation inputs.
/// Example: `build_ruled_surface_definition(&inputs)?`.
pub fn build_ruled_surface_definition(
    inputs: &RuledSurfaceInputs,
) -> Result<RuledSurfaceDefinition, Error> {
    let recipe = RuledSurfaceRecipe {
        version: 1,
        target_body: inputs.target_body.clone(),
        edges: inputs.edges.clone(),
        ruled_type: inputs.ruled_type,
        distance: inputs.distance,
        angle_deg: inputs.angle_deg,
        flip: inputs.flip,
        automatic_edge_chain: inputs.automatic_edge_chain,
        vector: inputs.vector.clone(),
    };
    RuledSurfaceDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Ruled Surface (`Ruled Surface1`) feature without mutating the document.
/// Example: `prepare_ruled_surface_definition(snapshot, &inputs)?`.
pub fn prepare_ruled_surface_definition(
    snapshot: &DocumentSnapshot,
    inputs: &RuledSurfaceInputs,
) -> Result<PreparedRuledSurface, Error> {
    let definition = build_ruled_surface_definition(inputs)?;
    Ok(PreparedRuledSurface {
        name: crate::fillet::next_name(snapshot, "Ruled Surface")?,
        definition,
    })
}

/// Replace an existing Ruled Surface feature's definition inside a transaction/preview candidate.
/// Example: `replace_ruled_surface(candidate, id, definition)?`.
pub fn replace_ruled_surface(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: RuledSurfaceDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_ruled_surface(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
