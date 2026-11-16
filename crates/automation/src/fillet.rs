// SPDX-License-Identifier: GPL-2.0-only
//! Prepare one validated fillet request for preview and commit (#4045).
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{FeatureId, fillet::FilletDefinition},
    history::{DocumentSnapshot, RecipeReservations},
};
mod certified;
mod wire;
pub use certified::{CertifiedFilletAccess, CertifiedFilletValidation};
pub(crate) use wire::apply as apply_wire;

/// Exact body resolution and validation for typed Fillet commands.
pub trait FilletAccess {
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
        name: next_name(snapshot)?,
        definition,
    })
}

fn next_name(snapshot: &DocumentSnapshot) -> Result<String, Error> {
    let limit = snapshot
        .work_geometry
        .features()
        .len()
        .checked_add(1)
        .ok_or_else(|| Error("feature count: expected room for another fillet name".into()))?;
    (1..=limit)
        .map(|index| format!("Fillet{index}"))
        .find(|name| {
            snapshot
                .work_geometry
                .features()
                .iter()
                .all(|feature| feature.name() != name)
        })
        .ok_or_else(|| {
            Error(format!(
                "fillet names through {limit}: expected an available name"
            ))
        })
}

#[cfg(test)]
#[path = "../tests/fillet/mod.rs"]
mod tests;
