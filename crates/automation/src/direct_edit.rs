// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Direct Edit creation and editing through the certified replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        direct_edit::{DirectEditDefinition, DirectEditOperation, DirectEditRecipe},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Direct Edit feature.
#[derive(Clone, Debug, PartialEq)]
pub struct DirectEditFeatureInputs {
    pub body: Vec<u8>,
    pub operations: Vec<DirectEditOperation>,
}

/// Validated Direct Edit recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedDirectEdit {
    name: String,
    definition: DirectEditDefinition,
}

impl PreparedDirectEdit {
    /// Inspect the validated Direct Edit definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &DirectEditDefinition {
        &self.definition
    }

    /// Add the prepared Direct Edit feature to a candidate snapshot; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_direct_edit(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Direct Edit definition from typed inputs.
/// Example: `build_direct_edit_definition(&inputs)?`.
pub fn build_direct_edit_definition(
    inputs: &DirectEditFeatureInputs,
) -> Result<DirectEditDefinition, Error> {
    let recipe = DirectEditRecipe {
        version: 1,
        body: inputs.body.clone(),
        operations: inputs.operations.clone(),
    };
    DirectEditDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Direct Edit feature without mutating the document.
/// Example: `prepare_direct_edit_definition(snapshot, &inputs)?`.
pub fn prepare_direct_edit_definition(
    snapshot: &DocumentSnapshot,
    inputs: &DirectEditFeatureInputs,
) -> Result<PreparedDirectEdit, Error> {
    let definition = build_direct_edit_definition(inputs)?;
    Ok(PreparedDirectEdit {
        name: crate::fillet::next_name(snapshot, "Direct Edit")?,
        definition,
    })
}

/// Replace an existing Direct Edit feature's definition inside a transaction or preview candidate.
/// Example: `replace_direct_edit(candidate, id, definition)?`.
pub fn replace_direct_edit(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: DirectEditDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_direct_edit(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
