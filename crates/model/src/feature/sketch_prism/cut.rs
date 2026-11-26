// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchPrismCause, SketchPrismDefinition, SketchPrismError};
use crate::feature::{FeatureId, fillet::reference_identity};

/// An extrusion Cut target identified by exact body name.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchPrismCutTarget {
    key: Vec<u8>,
    source: FeatureId,
}

impl SketchPrismCutTarget {
    /// Read the authored body name. Example: `target.key()` during exact replay.
    pub fn key(&self) -> &[u8] {
        &self.key
    }
    /// Read the original producing feature. Example: `target.source()` for dependency tracking.
    pub fn source(&self) -> FeatureId {
        self.source
    }
}

impl SketchPrismDefinition {
    /// Store the body selected for Cut. Example: `definition.with_cut_target(body_key)?`.
    pub fn with_cut_target(mut self, key: Vec<u8>) -> Result<Self, SketchPrismError> {
        let (_, source) = reference_identity(&key, 4, "extrude Cut target")
            .map_err(|cause| self.decline(SketchPrismCause::Definition(cause)))?;
        self.cut_target = Some(SketchPrismCutTarget { key, source });
        Ok(self)
    }
    /// Read the target for Cut. Example: `definition.cut_target()` before selecting a replay path.
    pub fn cut_target(&self) -> Option<&SketchPrismCutTarget> {
        self.cut_target.as_ref()
    }
}
