// SPDX-License-Identifier: GPL-2.0-only
//! A durable semantic face name and its producing feature.
use crate::{
    feature::FeatureId,
    identity::{ReferenceEnvelope, SemanticReferenceName},
};

/// A complete face name; binding to a current body occurs during replay.
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(try_from = "Vec<u8>", into = "Vec<u8>")]
pub struct WorkPlaneFaceReference {
    key: Vec<u8>,
    source: FeatureId,
}
impl WorkPlaneFaceReference {
    /// Validate the semantic face name; example: `WorkPlaneFaceReference::new(key)?`.
    pub fn new(key: Vec<u8>) -> Result<Self, String> {
        SemanticReferenceName::decode(&key)?;
        let envelope = ReferenceEnvelope::decode(&key).map_err(|cause| cause.to_string())?;
        if envelope.kind != 1 {
            return Err(format!(
                "face key {key:?} kind {}: expected face kind 1",
                envelope.kind
            ));
        }
        Ok(Self {
            source: FeatureId(envelope.context),
            key,
        })
    }
    /// Borrow the full name; example: `reference.key()` before resolving a face.
    pub fn key(&self) -> &[u8] {
        &self.key
    }
    /// Identify the producer; example: `reference.source()` for dependency ordering.
    pub fn source(&self) -> FeatureId {
        self.source
    }
}
impl TryFrom<Vec<u8>> for WorkPlaneFaceReference {
    type Error = String;
    fn try_from(key: Vec<u8>) -> Result<Self, Self::Error> {
        Self::new(key)
    }
}
impl From<WorkPlaneFaceReference> for Vec<u8> {
    fn from(reference: WorkPlaneFaceReference) -> Self {
        reference.key
    }
}
