// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Draft creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        draft::{DraftAngleMode, DraftDefinition, DraftMode, DraftPartingTool, DraftRecipe},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Draft.
#[derive(Clone, Debug, PartialEq)]
pub struct DraftFeatureInputs {
    pub body: Vec<u8>,
    pub mode: DraftMode,
    pub pull_direction: Vec<u8>,
    pub parting_tool: Option<DraftPartingTool>,
    pub faces: Vec<Vec<u8>>,
    pub angle_deg: f64,
    pub angle_mode: DraftAngleMode,
    pub direction_flipped: bool,
    pub automatic_blending: bool,
}

/// A validated Draft recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedDraft {
    name: String,
    definition: DraftDefinition,
}

impl PreparedDraft {
    /// Inspect the validated Draft definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &DraftDefinition {
        &self.definition
    }

    /// Add the prepared Draft recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_draft(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Draft definition from desktop/automation inputs.
pub fn build_draft_definition(inputs: &DraftFeatureInputs) -> Result<DraftDefinition, Error> {
    let recipe = DraftRecipe {
        version: 1,
        body: inputs.body.clone(),
        mode: inputs.mode,
        pull_direction: inputs.pull_direction.clone(),
        parting_tool: inputs.parting_tool.clone(),
        faces: inputs.faces.clone(),
        angle_deg: inputs.angle_deg,
        angle_mode: inputs.angle_mode,
        direction_flipped: inputs.direction_flipped,
        automatic_blending: inputs.automatic_blending,
    };
    let definition = DraftDefinition::from_recipe(recipe).map_err(Error)?;
    definition.resolve().map_err(Error)?;
    Ok(definition)
}

/// Prepare a new Draft feature (named `FaceDraft1`, `FaceDraft2`, ...) without mutating the document.
pub fn prepare_draft_definition(
    snapshot: &DocumentSnapshot,
    inputs: &DraftFeatureInputs,
) -> Result<PreparedDraft, Error> {
    let definition = build_draft_definition(inputs)?;
    Ok(PreparedDraft {
        name: crate::fillet::next_name(snapshot, "FaceDraft")?,
        definition,
    })
}

/// Replace an existing Draft feature's definition inside a transaction/preview candidate.
pub fn replace_draft(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: DraftDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_draft(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
