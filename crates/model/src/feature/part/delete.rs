// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    feature::{FeatureId, FeatureReadFootprint, FeatureReadKey},
    identity::{ReferenceEnvelope, SemanticReferenceName},
};

/// A body-set edit retains the complete name, never a transient body index (Go #1078).
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DeleteBodyDefinition {
    key: Vec<u8>,
    source: FeatureId,
}
impl DeleteBodyDefinition {
    /// Validate the stored name; current binding is checked at replay. Example: `DeleteBodyDefinition::new(key)?`.
    pub fn new(key: Vec<u8>) -> Result<Self, String> {
        SemanticReferenceName::decode(&key)?;
        let envelope = ReferenceEnvelope::decode(&key).map_err(|cause| cause.to_string())?;
        if envelope.kind != 4 {
            return Err(format!(
                "delete-body key {key:?} kind {}: expected body kind 4",
                envelope.kind
            ));
        }
        let source = FeatureId(envelope.context);
        Ok(Self { key, source })
    }
    /// Borrow the exact target bytes. Example: `definition.key()` for owner validation.
    pub fn key(&self) -> &[u8] {
        &self.key
    }
    /// Read the producing feature without geometric matching. Example: `definition.source()`.
    pub fn source(&self) -> FeatureId {
        self.source
    }
    pub(crate) fn reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new([FeatureReadKey::Feature(self.source)], false)
    }
}
