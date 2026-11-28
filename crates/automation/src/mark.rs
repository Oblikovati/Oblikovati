// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Mark creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        mark::{MarkDefinition, MarkDirection, MarkGeometrySet, MarkMethod, MarkRecipe},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Mark feature.
#[derive(Clone, Debug, PartialEq)]
pub struct MarkFeatureInputs {
    pub body: Vec<u8>,
    pub geometry_sets: Vec<MarkGeometrySet>,
    pub method: MarkMethod,
    pub faces: Vec<Vec<u8>>,
    pub automatic_face_chain: bool,
    pub direction: MarkDirection,
}

/// A validated Mark recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedMark {
    name: String,
    definition: MarkDefinition,
}

impl PreparedMark {
    /// Inspect the validated Mark definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &MarkDefinition {
        &self.definition
    }

    /// Add the prepared Mark recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_mark(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Mark definition from desktop/automation inputs.
pub fn build_mark_definition(inputs: &MarkFeatureInputs) -> Result<MarkDefinition, Error> {
    let recipe = MarkRecipe {
        version: 1,
        body: inputs.body.clone(),
        geometry_sets: inputs.geometry_sets.clone(),
        method: inputs.method,
        faces: inputs.faces.clone(),
        automatic_face_chain: inputs.automatic_face_chain,
        direction: inputs.direction,
    };
    MarkDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Mark feature without mutating the document.
pub fn prepare_mark_definition(
    snapshot: &DocumentSnapshot,
    inputs: &MarkFeatureInputs,
) -> Result<PreparedMark, Error> {
    let definition = build_mark_definition(inputs)?;
    Ok(PreparedMark {
        name: crate::fillet::next_name(snapshot, "Mark")?,
        definition,
    })
}

/// Replace an existing Mark feature's definition inside a transaction/preview candidate.
pub fn replace_mark(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: MarkDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_mark(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
