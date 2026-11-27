// SPDX-License-Identifier: GPL-2.0-only
//! A named edge center or vertex belongs to its producing feature (#4107).
use crate::{
    feature::FeatureId,
    identity::{ReferenceEnvelope, topology_reference_document},
};

/// Stable topology identity used as a sketch-frame origin.
/// Example: `SketchFrameTopologyOrigin::new(key)?`.
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(try_from = "Vec<u8>", into = "Vec<u8>")]
pub struct SketchFrameTopologyOrigin {
    key: Vec<u8>,
    source: FeatureId,
}
impl SketchFrameTopologyOrigin {
    /// Validate a named edge or vertex; example: `SketchFrameTopologyOrigin::new(key)?`.
    pub fn new(key: Vec<u8>) -> Result<Self, String> {
        topology_reference_document(&key)?;
        let envelope = ReferenceEnvelope::decode(&key).map_err(|error| error.to_string())?;
        if !matches!(envelope.kind, 2 | 3) || envelope.context == 0 {
            return Err(format!(
                "frame origin key {key:?}, kind {}, source {}: expected a named edge or vertex with a nonzero feature",
                envelope.kind, envelope.context
            ));
        }
        Ok(Self {
            source: FeatureId(envelope.context),
            key,
        })
    }
    /// Borrow the durable name; example: `reference.key()` before binding it.
    pub fn key(&self) -> &[u8] {
        &self.key
    }
    /// Read its producer; example: `reference.source()` for replay ordering.
    pub fn source(&self) -> FeatureId {
        self.source
    }
}
impl TryFrom<Vec<u8>> for SketchFrameTopologyOrigin {
    type Error = String;
    fn try_from(key: Vec<u8>) -> Result<Self, Self::Error> {
        Self::new(key)
    }
}
impl From<SketchFrameTopologyOrigin> for Vec<u8> {
    fn from(reference: SketchFrameTopologyOrigin) -> Vec<u8> {
        reference.key
    }
}
