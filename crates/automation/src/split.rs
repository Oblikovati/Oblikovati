// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Split creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        split::{SplitDefinition, SplitKeepSide, SplitMode, SplitRecipe, SplitTarget, SplitTool},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Split feature.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SplitFeatureInputs {
    pub mode: SplitMode,
    pub tool: SplitTool,
    pub target: SplitTarget,
    pub keep_side: SplitKeepSide,
}

/// A validated Split recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedSplit {
    name: String,
    definition: SplitDefinition,
}

impl PreparedSplit {
    /// Inspect the validated Split definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &SplitDefinition {
        &self.definition
    }

    /// Add the prepared Split recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_split(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Split definition from desktop/automation inputs.
/// Example: `build_split_definition(&inputs)?`.
pub fn build_split_definition(inputs: &SplitFeatureInputs) -> Result<SplitDefinition, Error> {
    let recipe = SplitRecipe {
        version: 1,
        mode: inputs.mode,
        tool: inputs.tool.clone(),
        target: inputs.target.clone(),
        keep_side: inputs.keep_side,
    };
    SplitDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Split feature (`Split1`, `Split2`, ...) without mutating the document.
/// Example: `prepare_split_definition(snapshot, &inputs)?`.
pub fn prepare_split_definition(
    snapshot: &DocumentSnapshot,
    inputs: &SplitFeatureInputs,
) -> Result<PreparedSplit, Error> {
    let definition = build_split_definition(inputs)?;
    Ok(PreparedSplit {
        name: crate::fillet::next_name(snapshot, "Split")?,
        definition,
    })
}

/// Replace an existing Split feature's definition inside a transaction/preview candidate.
/// Example: `replace_split(candidate, id, definition)?`.
pub fn replace_split(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SplitDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_split(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
