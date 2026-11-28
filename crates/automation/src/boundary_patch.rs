// SPDX-License-Identifier: GPL-2.0-only
//! Typed Surface Boundary Patch creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        boundary_patch::{
            BoundaryCurveRef, BoundaryEdgeSet, BoundaryPatchDefinition, BoundaryPatchRecipe,
        },
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Surface Boundary Patch feature.
#[derive(Clone, Debug, PartialEq)]
pub struct BoundaryPatchInputs {
    pub automatic_edge_chain: bool,
    pub edge_sets: Vec<BoundaryEdgeSet>,
    pub guide_rails: Vec<BoundaryCurveRef>,
}

/// A validated Boundary Patch recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedBoundaryPatch {
    name: String,
    definition: BoundaryPatchDefinition,
}

impl PreparedBoundaryPatch {
    /// Inspect the generated feature name (`Boundary Patch1`, `Boundary Patch2`, ...).
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Inspect the validated Boundary Patch definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &BoundaryPatchDefinition {
        &self.definition
    }

    /// Add the prepared Boundary Patch recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_boundary_patch(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Boundary Patch definition from desktop/automation inputs.
pub fn build_boundary_patch_definition(
    inputs: &BoundaryPatchInputs,
) -> Result<BoundaryPatchDefinition, Error> {
    let recipe = BoundaryPatchRecipe {
        version: 1,
        automatic_edge_chain: inputs.automatic_edge_chain,
        edge_sets: inputs.edge_sets.clone(),
        guide_rails: inputs.guide_rails.clone(),
    };
    BoundaryPatchDefinition::from_recipe(recipe).map_err(Error)
}

/// Prepare a new Boundary Patch feature (`Boundary Patch1`, `Boundary Patch2`, ...) without mutating the document.
pub fn prepare_boundary_patch_definition(
    snapshot: &DocumentSnapshot,
    inputs: &BoundaryPatchInputs,
) -> Result<PreparedBoundaryPatch, Error> {
    let definition = build_boundary_patch_definition(inputs)?;
    Ok(PreparedBoundaryPatch {
        name: crate::fillet::next_name(snapshot, "Boundary Patch")?,
        definition,
    })
}

/// Replace an existing Boundary Patch feature's definition inside a transaction/preview candidate.
pub fn replace_boundary_patch(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: BoundaryPatchDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_boundary_patch(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
