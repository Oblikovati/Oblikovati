// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Finish creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        finish::{FinishDefinition, FinishRecipe, FinishType},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Finish feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FinishFeatureInputs {
    pub body: Vec<u8>,
    pub included_faces: Vec<Vec<u8>>,
    pub excluded_faces: Vec<Vec<u8>>,
    pub finish_type: FinishType,
    pub process: String,
    pub appearance: String,
    pub thickness: String,
    pub hardness: String,
    pub depth: String,
    pub description: String,
    pub short_description: String,
    pub comment: String,
    pub export_all_parameters: bool,
}

/// A validated Finish recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedFinish {
    name: String,
    definition: FinishDefinition,
}

impl PreparedFinish {
    /// Inspect the validated Finish definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &FinishDefinition {
        &self.definition
    }

    /// Add the prepared Finish recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_finish(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Finish definition from desktop/automation inputs.
pub fn build_finish_definition(inputs: &FinishFeatureInputs) -> Result<FinishDefinition, Error> {
    let recipe = FinishRecipe {
        version: 1,
        body: inputs.body.clone(),
        included_faces: inputs.included_faces.clone(),
        excluded_faces: inputs.excluded_faces.clone(),
        finish_type: inputs.finish_type,
        process: inputs.process.clone(),
        appearance: inputs.appearance.clone(),
        thickness: inputs.thickness.clone(),
        hardness: inputs.hardness.clone(),
        depth: inputs.depth.clone(),
        description: inputs.description.clone(),
        short_description: inputs.short_description.clone(),
        comment: inputs.comment.clone(),
        export_all_parameters: inputs.export_all_parameters,
    };
    let definition = FinishDefinition::from_recipe(recipe).map_err(Error)?;
    definition.resolve().map_err(Error)?;
    Ok(definition)
}

/// Prepare a new Finish feature (named `Finish1`, `Finish2`, ...) without mutating the document.
pub fn prepare_finish_definition(
    snapshot: &DocumentSnapshot,
    inputs: &FinishFeatureInputs,
) -> Result<PreparedFinish, Error> {
    let definition = build_finish_definition(inputs)?;
    Ok(PreparedFinish {
        name: crate::fillet::next_name(snapshot, "Finish")?,
        definition,
    })
}

/// Replace an existing Finish feature's definition inside a transaction/preview candidate.
pub fn replace_finish(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: FinishDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_finish(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
