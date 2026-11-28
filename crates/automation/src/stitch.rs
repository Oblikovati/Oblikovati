// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Surface Stitch creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        stitch::{StitchDefinition, StitchRecipe},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Surface Stitch feature.
#[derive(Clone, Debug, PartialEq)]
pub struct StitchInputs {
    pub surfaces: Vec<Vec<u8>>,
    pub maintain_as_surface: bool,
    /// Maximum stitching tolerance in millimeters.
    pub maximum_tolerance_mm: f64,
    pub show_edge_conditions: bool,
    pub show_near_tangent: bool,
}

/// A validated Stitch recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedStitch {
    name: String,
    definition: StitchDefinition,
}

impl PreparedStitch {
    /// Inspect the generated feature name (`Stitch Surface1`, `Stitch Surface2`, ...).
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Inspect the validated Stitch definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &StitchDefinition {
        &self.definition
    }

    /// Add the prepared Stitch recipe to a private candidate.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_stitch(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Stitch definition from desktop/automation inputs.
/// Example: `build_stitch_definition(&inputs)?`.
pub fn build_stitch_definition(inputs: &StitchInputs) -> Result<StitchDefinition, Error> {
    let recipe = StitchRecipe {
        version: 1,
        surfaces: inputs.surfaces.clone(),
        maintain_as_surface: inputs.maintain_as_surface,
        maximum_tolerance_mm: inputs.maximum_tolerance_mm,
        show_edge_conditions: inputs.show_edge_conditions,
        show_near_tangent: inputs.show_near_tangent,
    };
    StitchDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Stitch (`Stitch Surface1`) feature without mutating the document.
/// Example: `prepare_stitch_definition(snapshot, &inputs)?`.
pub fn prepare_stitch_definition(
    snapshot: &DocumentSnapshot,
    inputs: &StitchInputs,
) -> Result<PreparedStitch, Error> {
    let definition = build_stitch_definition(inputs)?;
    Ok(PreparedStitch {
        name: crate::fillet::next_name(snapshot, "Stitch Surface")?,
        definition,
    })
}

/// Replace an existing Stitch feature's definition inside a transaction/preview candidate.
/// Example: `replace_stitch(candidate, id, definition)?`.
pub fn replace_stitch(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: StitchDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_stitch(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
