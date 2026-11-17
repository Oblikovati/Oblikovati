// SPDX-License-Identifier: GPL-2.0-only
//! Authored named-edge fillet inputs, independent of a kernel result (#4045).
use super::{FeatureId, FeatureReadFootprint, FeatureReadKey, work_geometry::DatumLength};
use crate::{
    identity::{DocumentGuid, ReferenceEnvelope, topology_reference_document},
    parameter::ParameterCollection,
};
use std::collections::BTreeSet;

/// Authored named-edge blend input; geometry and bindings are resolved during replay.
#[derive(Clone, Debug, PartialEq)]
pub struct BlendDefinition {
    body: Vec<u8>,
    body_source: FeatureId,
    edges: Vec<Vec<u8>>,
    radius: DatumLength,
    sources: Vec<FeatureId>,
}

/// Compatibility name for authored fillet inputs.
pub type FilletDefinition = BlendDefinition;

/// Versioned fillet input without a cached kernel result.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct FilletRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub radius: DatumLength,
}

impl BlendDefinition {
    /// Validate names and sources; current body membership is checked during replay.
    /// Example: `FilletDefinition::new(body_key, edge_keys, radius)?`.
    pub fn new(body: Vec<u8>, edges: Vec<Vec<u8>>, radius: DatumLength) -> Result<Self, String> {
        Self::new_blend(body, edges, radius, "fillet")
    }
    pub(super) fn new_blend(
        body: Vec<u8>,
        edges: Vec<Vec<u8>>,
        radius: DatumLength,
        operation: &str,
    ) -> Result<Self, String> {
        let (document, source) = reference_identity(&body, 4, operation)?;
        validate_radius_source(radius, operation)?;
        let (edges, sources) = validate_edges(document, source, edges, operation)?;
        Ok(Self {
            body,
            body_source: source,
            edges,
            radius,
            sources,
        })
    }
    /// Inspect the body reference; example: `definition.body()` before current binding.
    pub fn body(&self) -> &[u8] {
        &self.body
    }
    /// Read the named body's producer; example: `definition.body_source()` for replay.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }
    /// Inspect canonical edge names; example: `definition.edges()` for replay.
    pub fn edges(&self) -> &[Vec<u8>] {
        &self.edges
    }
    /// Inspect the authored scalar; example: `definition.radius()` for a parameter editor.
    pub fn radius(&self) -> DatumLength {
        self.radius
    }
    /// Capture authored inputs; example: `serde_json::to_string(&definition.recipe())?`.
    pub fn recipe(&self) -> FilletRecipe {
        FilletRecipe {
            version: 1,
            body: self.body.clone(),
            edges: self.edges.clone(),
            radius: self.radius,
        }
    }
    /// Restore validated authored inputs; example: `FilletDefinition::from_recipe(&recipe)?`.
    pub fn from_recipe(recipe: &FilletRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "fillet recipe version {}: expected version 1",
                recipe.version
            ));
        }
        Self::new(recipe.body.clone(), recipe.edges.clone(), recipe.radius)
    }
    /// Attribute all source features and the radius parameter; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let features = self.sources.iter().copied().map(FeatureReadKey::Feature);
        let parameter = self.radius.parameter().map(FeatureReadKey::Parameter);
        FeatureReadFootprint::new(features.chain(parameter), false)
    }
    /// Resolve a positive current radius; example: `definition.resolved_radius(&parameters)?`.
    pub fn resolved_radius(&self, parameters: &ParameterCollection<u64>) -> Result<f64, String> {
        self.resolved_blend_length(parameters, "fillet")
    }
    pub(super) fn resolved_blend_length(
        &self,
        parameters: &ParameterCollection<u64>,
        operation: &str,
    ) -> Result<f64, String> {
        let value = self
            .radius
            .resolve(parameters)
            .map_err(|cause| cause.to_string())?;
        if value <= 0. {
            return Err(format!(
                "{operation} length {value:?}: expected a finite positive length"
            ));
        }
        Ok(value)
    }
}

fn reference_identity(
    key: &[u8],
    expected: u32,
    operation: &str,
) -> Result<(DocumentGuid, FeatureId), String> {
    let document = topology_reference_document(key).map_err(|cause| {
        format!("{operation} key {key:?}: expected a semantic reference: {cause}")
    })?;
    let envelope = ReferenceEnvelope::decode(key).map_err(|cause| cause.to_string())?;
    if envelope.kind != expected {
        return Err(format!(
            "{operation} key {key:?} kind {}: expected kind {expected}",
            envelope.kind
        ));
    }
    Ok((document, FeatureId(envelope.context)))
}

fn validate_edges(
    document: DocumentGuid,
    body: FeatureId,
    edges: Vec<Vec<u8>>,
    operation: &str,
) -> Result<(Vec<Vec<u8>>, Vec<FeatureId>), String> {
    if edges.is_empty() {
        return Err(format!(
            "{operation} edges []: expected at least one named edge"
        ));
    }
    let mut unique = BTreeSet::new();
    let mut sources = BTreeSet::from([body]);
    for key in edges {
        let (edge_document, source) = reference_identity(&key, 2, operation)?;
        if edge_document != document || !unique.insert(key.clone()) {
            return Err(format!(
                "{operation} edge {key:?}: expected one occurrence in document {document:?}"
            ));
        }
        sources.insert(source);
    }
    Ok((unique.into_iter().collect(), sources.into_iter().collect()))
}

fn validate_radius_source(radius: DatumLength, operation: &str) -> Result<(), String> {
    radius
        .validate_source()
        .map_err(|cause| cause.to_string())?;
    if matches!(radius, DatumLength::Constant(value) if value <= 0.)
        || radius.parameter() == Some(0)
    {
        return Err(format!(
            "{operation} length source {radius:?}: expected a positive constant or nonzero parameter"
        ));
    }
    Ok(())
}
