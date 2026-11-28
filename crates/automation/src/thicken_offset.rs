// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Thicken/Offset creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        thicken_offset::{
            ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean,
            ThickenDirection, ThickenOffsetDefinition, ThickenOffsetRecipe, ThickenOutputMode,
        },
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Thicken/Offset feature.
#[derive(Clone, Debug, PartialEq)]
pub struct ThickenOffsetInputs {
    pub target_body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    pub output: ThickenOutputMode,
    /// Offset or thicken distance in model units (cm).
    pub distance: f64,
    pub direction: ThickenDirection,
    pub boolean: ThickenBoolean,
    pub automatic_face_chain: bool,
    pub automatic_blending: bool,
    pub allow_approximation: bool,
    pub approximation_type: ThickenApproximationType,
    pub approximation_tolerance: ThickenApproximationTolerance,
    pub approximation_percent: Option<f64>,
}

/// A validated Thicken/Offset recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedThickenOffset {
    name: String,
    definition: ThickenOffsetDefinition,
}

impl PreparedThickenOffset {
    /// Inspect the generated feature name (`Thicken1` or `OffsetSrf1`).
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Inspect the validated Thicken/Offset definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &ThickenOffsetDefinition {
        &self.definition
    }

    /// Add the prepared Thicken/Offset recipe to a private candidate.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_thicken_offset(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Thicken/Offset definition from desktop/automation inputs.
/// Example: `build_thicken_offset_definition(&inputs)?`.
pub fn build_thicken_offset_definition(
    inputs: &ThickenOffsetInputs,
) -> Result<ThickenOffsetDefinition, Error> {
    let recipe = ThickenOffsetRecipe {
        version: 1,
        target_body: inputs.target_body.clone(),
        faces: inputs.faces.clone(),
        output: inputs.output,
        distance: inputs.distance,
        direction: inputs.direction,
        boolean: inputs.boolean,
        automatic_face_chain: inputs.automatic_face_chain,
        automatic_blending: inputs.automatic_blending,
        allow_approximation: inputs.allow_approximation,
        approximation_type: inputs.approximation_type,
        approximation_tolerance: inputs.approximation_tolerance,
        approximation_percent: inputs.approximation_percent,
    };
    ThickenOffsetDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Thicken (`Thicken1`) or Offset Surface (`OffsetSrf1`) feature without mutating the document.
/// Example: `prepare_thicken_offset_definition(snapshot, &inputs)?`.
pub fn prepare_thicken_offset_definition(
    snapshot: &DocumentSnapshot,
    inputs: &ThickenOffsetInputs,
) -> Result<PreparedThickenOffset, Error> {
    let definition = build_thicken_offset_definition(inputs)?;
    let prefix = match inputs.output {
        ThickenOutputMode::Solid => "Thicken",
        ThickenOutputMode::Surface => "OffsetSrf",
    };
    Ok(PreparedThickenOffset {
        name: crate::fillet::next_name(snapshot, prefix)?,
        definition,
    })
}

/// Replace an existing Thicken/Offset feature's definition inside a transaction/preview candidate.
/// Example: `replace_thicken_offset(candidate, id, definition)?`.
pub fn replace_thicken_offset(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: ThickenOffsetDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_thicken_offset(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
