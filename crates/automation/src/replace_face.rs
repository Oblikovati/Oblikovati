// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Surface Replace Face creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        replace_face::{ReplaceFaceDefinition, ReplaceFaceRecipe, ReplaceFaceTarget},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Surface Replace Face feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ReplaceFaceFeatureInputs {
    pub body: Vec<u8>,
    pub existing_faces: Vec<Vec<u8>>,
    pub new_faces: Vec<ReplaceFaceTarget>,
    pub automatic_face_chain: bool,
}

/// A validated Replace Face recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedReplaceFace {
    name: String,
    definition: ReplaceFaceDefinition,
}

impl PreparedReplaceFace {
    /// Inspect the validated Replace Face definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &ReplaceFaceDefinition {
        &self.definition
    }

    /// Add the prepared Replace Face recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_replace_face(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Replace Face definition from desktop/automation inputs.
pub fn build_replace_face_definition(
    inputs: &ReplaceFaceFeatureInputs,
) -> Result<ReplaceFaceDefinition, Error> {
    let recipe = ReplaceFaceRecipe {
        version: 1,
        body: inputs.body.clone(),
        existing_faces: inputs.existing_faces.clone(),
        new_faces: inputs.new_faces.clone(),
        automatic_face_chain: inputs.automatic_face_chain,
    };
    let definition = ReplaceFaceDefinition::from_recipe(recipe).map_err(Error)?;
    definition.resolve().map_err(Error)?;
    Ok(definition)
}

/// Prepare a new Replace Face feature (named `Replace Face1`, `Replace Face2`, ...) without mutating the document.
pub fn prepare_replace_face_definition(
    snapshot: &DocumentSnapshot,
    inputs: &ReplaceFaceFeatureInputs,
) -> Result<PreparedReplaceFace, Error> {
    let definition = build_replace_face_definition(inputs)?;
    Ok(PreparedReplaceFace {
        name: crate::fillet::next_name(snapshot, "Replace Face")?,
        definition,
    })
}

/// Replace an existing Replace Face feature's definition inside a transaction/preview candidate.
pub fn replace_replace_face(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: ReplaceFaceDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_replace_face(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
