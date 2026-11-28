// SPDX-License-Identifier: GPL-2.0-only
//! Typed Part Modify Thread creation and editing through the certified part replay owner (#3918).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        thread::{ThreadDefinition, ThreadRecipe, ThreadSpecification},
        work_geometry::DatumLength,
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// Typed desktop/automation inputs for creating or editing a Part Modify Thread.
#[derive(Clone, Debug, PartialEq)]
pub struct ThreadFeatureInputs {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub full_depth: bool,
    pub depth: Option<String>,
    pub offset: Option<String>,
    pub display_in_model: bool,
    pub spec: ThreadSpecification,
}

/// A validated Thread recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedThread {
    name: String,
    definition: ThreadDefinition,
}

impl PreparedThread {
    /// Inspect the validated Thread definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &ThreadDefinition {
        &self.definition
    }

    /// Add the prepared Thread recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_thread(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Build a validated Thread definition from desktop/automation inputs.
pub fn build_thread_definition(
    snapshot: &DocumentSnapshot,
    inputs: &ThreadFeatureInputs,
) -> Result<ThreadDefinition, Error> {
    let depth = if inputs.full_depth {
        match inputs.depth.as_deref().map(str::trim) {
            Some(expr) if !expr.is_empty() => {
                crate::extrusion::positive_length(snapshot, expr, "thread depth").ok()
            }
            _ => None,
        }
    } else {
        let expr = inputs
            .depth
            .as_deref()
            .ok_or_else(|| Error("thread depth: expected a positive length".into()))?;
        Some(crate::extrusion::positive_length(
            snapshot,
            expr,
            "thread depth",
        )?)
    };
    let offset = if inputs.full_depth {
        None
    } else {
        match inputs.offset.as_deref().map(str::trim) {
            Some(expr) if !expr.is_empty() => Some(nonnegative_length(snapshot, expr)?),
            _ => Some(DatumLength::Constant(0.0)),
        }
    };
    let recipe = ThreadRecipe {
        version: 1,
        body: inputs.body.clone(),
        face: inputs.face.clone(),
        full_depth: inputs.full_depth,
        depth,
        offset,
        display_in_model: inputs.display_in_model,
        spec: inputs.spec.clone(),
    };
    let definition = ThreadDefinition::from_recipe(recipe).map_err(Error)?;
    definition.resolve(&snapshot.parameters).map_err(Error)?;
    Ok(definition)
}

/// Prepare a new Thread feature without mutating the document.
pub fn prepare_thread_definition(
    snapshot: &DocumentSnapshot,
    inputs: &ThreadFeatureInputs,
) -> Result<PreparedThread, Error> {
    let definition = build_thread_definition(snapshot, inputs)?;
    Ok(PreparedThread {
        name: crate::fillet::next_name(snapshot, "Thread")?,
        definition,
    })
}

/// Replace an existing Thread feature's definition inside a transaction/preview candidate.
pub fn replace_thread(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: ThreadDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_thread(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}

fn nonnegative_length(snapshot: &DocumentSnapshot, source: &str) -> Result<DatumLength, Error> {
    let length = crate::extrusion::finite_length(snapshot, source, "thread offset")?;
    let value = length
        .resolve(&snapshot.parameters)
        .map_err(|cause| Error(cause.to_string()))?;
    if !value.is_finite() || value < 0.0 {
        return Err(Error(format!(
            "thread offset {source:?} resolved to {value:?}: expected a finite nonnegative length"
        )));
    }
    Ok(length)
}
