// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Delete Face creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        delete_face::{DeleteFaceDefinition, DeleteFaceRecipe},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Delete Face.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DeleteFaceFeatureInputs {
    pub body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    pub heal_remaining_faces: bool,
    pub select_lump_or_void: bool,
}

/// A validated Delete Face recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedDeleteFace {
    name: String,
    definition: DeleteFaceDefinition,
}

impl PreparedDeleteFace {
    /// Inspect the validated Delete Face definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &DeleteFaceDefinition {
        &self.definition
    }

    /// Add the prepared Delete Face recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_delete_face(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Delete Face definition from desktop/automation inputs.
pub fn build_delete_face_definition(
    inputs: &DeleteFaceFeatureInputs,
) -> Result<DeleteFaceDefinition, Error> {
    let recipe = DeleteFaceRecipe {
        version: 1,
        body: inputs.body.clone(),
        faces: inputs.faces.clone(),
        heal_remaining_faces: inputs.heal_remaining_faces,
        select_lump_or_void: inputs.select_lump_or_void,
    };
    let definition = DeleteFaceDefinition::from_recipe(recipe).map_err(Error)?;
    definition.resolve().map_err(Error)?;
    Ok(definition)
}

/// Prepare a new Delete Face feature (named `Delete Face1`, `Delete Face2`, ...) without mutating the document.
pub fn prepare_delete_face_definition(
    snapshot: &DocumentSnapshot,
    inputs: &DeleteFaceFeatureInputs,
) -> Result<PreparedDeleteFace, Error> {
    let definition = build_delete_face_definition(inputs)?;
    Ok(PreparedDeleteFace {
        name: crate::fillet::next_name(snapshot, "Delete Face")?,
        definition,
    })
}

/// Replace an existing Delete Face feature's definition inside a transaction/preview candidate.
pub fn replace_delete_face(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: DeleteFaceDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_delete_face(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
