// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Combine creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        combine::{CombineDefinition, CombineOperation, CombineRecipe},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Combine feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CombineInputs {
    pub base_body: Vec<u8>,
    pub tool_bodies: Vec<Vec<u8>>,
    pub operation: CombineOperation,
    pub keep_toolbodies: bool,
}

/// A validated Combine recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedCombine {
    name: String,
    definition: CombineDefinition,
}

impl PreparedCombine {
    /// Inspect the validated Combine definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &CombineDefinition {
        &self.definition
    }

    /// Add the prepared Combine recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_combine(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Combine definition from desktop/automation inputs.
/// Example: `build_combine_definition(&inputs)?`.
pub fn build_combine_definition(inputs: &CombineInputs) -> Result<CombineDefinition, Error> {
    let recipe = CombineRecipe {
        version: 1,
        base_body: inputs.base_body.clone(),
        tool_bodies: inputs.tool_bodies.clone(),
        operation: inputs.operation,
        keep_toolbodies: inputs.keep_toolbodies,
    };
    CombineDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Combine feature without mutating the document.
/// Example: `prepare_combine_definition(snapshot, &inputs)?`.
pub fn prepare_combine_definition(
    snapshot: &DocumentSnapshot,
    inputs: &CombineInputs,
) -> Result<PreparedCombine, Error> {
    let definition = build_combine_definition(inputs)?;
    Ok(PreparedCombine {
        name: crate::fillet::next_name(snapshot, "Combine")?,
        definition,
    })
}

/// Replace an existing Combine feature's definition inside a transaction/preview candidate.
/// Example: `replace_combine(candidate, id, definition)?`.
pub fn replace_combine(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: CombineDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_combine(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
