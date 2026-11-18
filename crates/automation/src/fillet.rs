// SPDX-License-Identifier: GPL-2.0-only
//! Prepare one validated fillet request for preview and commit (#4045).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{FeatureId, fillet::FilletDefinition},
    history::{DocumentSnapshot, RecipeReservations},
};
mod certified;
mod wire;
pub(crate) use certified::resolve_blend_body;
pub use certified::{CertifiedFilletAccess, CertifiedFilletValidation};
pub(crate) use wire::apply as apply_wire;

pub(crate) fn blend_input_body<O, W, F>(
    files: &mut oblikovati_session::files::FileWorkspace<O, W, F>,
    backend: &dyn FilletAccess,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    document: u64,
    edges: &[Vec<u8>],
) -> Result<Vec<u8>, Error>
where
    O: oblikovati_session::TransactionObserver,
    W: oblikovati_session::workspace::WorkspaceObserver,
    F: oblikovati_persistence::file_access::DocumentFiles,
{
    if let Some(replay) = replay.filter(|owner| backend.accepts_policy(owner.resolution_policy())) {
        return replay.blend_body(files, document, edges).map_err(Error);
    }
    let workspace = files.workspace();
    let session = workspace
        .session(document)
        .map_err(|error| Error(error.to_string()))?;
    let guid = workspace
        .identity(document)
        .map_err(|error| Error(error.to_string()))?
        .internal_name;
    backend.body(session.history(), guid, edges).map_err(Error)
}

/// Exact body resolution and validation for typed Fillet commands.
pub trait FilletAccess {
    /// Require the validator's policy for reuse; example: `access.accepts_policy(policy)`.
    fn accepts_policy(&self, policy: Option<oblikovati_martensite::PrismResolutionPolicy>) -> bool;
    /// Resolve all selected keys to one current body; example: `access.body(history, document, &edges)?`.
    fn body(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
        document: oblikovati_model::identity::DocumentGuid,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String>;
    /// Validate a candidate and count its live bodies; example: `access.validate(document, snapshot, feature)?`.
    fn validate(
        &self,
        document: oblikovati_model::identity::DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String>;
    /// Validate a chamfer candidate; example: `access.validate_chamfer(document, candidate, feature)?`.
    fn validate_chamfer(
        &self,
        _document: oblikovati_model::identity::DocumentGuid,
        _candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        Err(format!(
            "chamfer {feature:?}: expected an available certified evaluator"
        ))
    }
}

/// Geometry validation performed before a fillet candidate is adopted.
pub trait FilletValidation {
    /// Validate a solved candidate; example: `validator.validate(candidate, feature)?`.
    fn validate(&self, candidate: &DocumentSnapshot, feature: FeatureId) -> Result<(), String>;
}

/// Canonical authored inputs shared by a preview and its eventual command.
#[derive(Clone)]
pub struct PreparedFillet {
    name: String,
    definition: FilletDefinition,
}
impl PreparedFillet {
    /// Append to a private transaction candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        candidate: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        candidate
            .create_fillet(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
    /// Validate the candidate before publication; example: `prepared.validate(&validator, candidate, feature)?`.
    pub fn validate(
        validator: &dyn FilletValidation,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<FeatureId, Error> {
        validator.validate(candidate, feature).map_err(Error)?;
        Ok(feature)
    }
}

/// Prepare an exact body, edge set and length expression; example: `prepare_fillet(snapshot, body, edges, "2 mm")?`.
pub fn prepare_fillet(
    snapshot: &DocumentSnapshot,
    body: Vec<u8>,
    edges: Vec<Vec<u8>>,
    radius: &str,
) -> Result<PreparedFillet, Error> {
    let radius = crate::extrusion::positive_length(snapshot, radius, "fillet radius")?;
    let definition = FilletDefinition::new(body, edges, radius).map_err(Error)?;
    Ok(PreparedFillet {
        name: next_name(snapshot, "Fillet")?,
        definition,
    })
}

pub(crate) fn next_name(snapshot: &DocumentSnapshot, prefix: &str) -> Result<String, Error> {
    let limit = snapshot
        .work_geometry
        .features()
        .len()
        .checked_add(1)
        .ok_or_else(|| {
            Error(format!(
                "feature count: expected room for another {prefix} name"
            ))
        })?;
    (1..=limit)
        .map(|index| format!("{prefix}{index}"))
        .find(|name| {
            snapshot
                .work_geometry
                .features()
                .iter()
                .all(|feature| feature.name() != name)
        })
        .ok_or_else(|| {
            Error(format!(
                "{prefix} names through {limit}: expected an available name"
            ))
        })
}

#[cfg(test)]
#[path = "../tests/fillet/mod.rs"]
mod tests;
