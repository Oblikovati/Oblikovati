// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchPrismCause, SketchPrismDefinition, SketchPrismError};
use crate::feature::{FeatureId, fillet::reference_identity};

/// An extrusion Boolean target identified by exact body name.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchPrismTarget {
    operation: PrismBoolean,
    key: Vec<u8>,
    source: FeatureId,
}

/// Boolean operations that replace a named extrusion target.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum PrismBoolean {
    Cut,
    Join,
    Intersect,
}

impl std::fmt::Display for PrismBoolean {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str(match self {
            Self::Cut => "Cut",
            Self::Join => "Join",
            Self::Intersect => "Intersect",
        })
    }
}

impl SketchPrismTarget {
    /// Read the authored operation. Example: `target.operation()`.
    pub fn operation(&self) -> PrismBoolean {
        self.operation
    }
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
    pub fn with_cut_target(self, key: Vec<u8>) -> Result<Self, SketchPrismError> {
        self.with_boolean_target(PrismBoolean::Cut, key)
    }

    /// Set the operation and exact body name. Example: `definition.with_boolean_target(PrismBoolean::Join, key)?`.
    pub fn with_boolean_target(
        mut self,
        operation: PrismBoolean,
        key: Vec<u8>,
    ) -> Result<Self, SketchPrismError> {
        let (_, source) = reference_identity(&key, 4, &format!("extrude {operation} target"))
            .map_err(|cause| self.decline(SketchPrismCause::Definition(cause)))?;
        if operation != PrismBoolean::Cut
            && matches!(
                self.extent(),
                Some(super::SketchPrismExtent::ThroughAll { .. })
            )
        {
            return Err(self.decline(SketchPrismCause::Definition(
                "through-all extent: expected operation Cut".into(),
            )));
        }
        self.boolean_target = Some(SketchPrismTarget {
            operation,
            key,
            source,
        });
        Ok(self)
    }
    /// Read the target for Cut. Example: `definition.cut_target()` before resolving Through All.
    pub fn cut_target(&self) -> Option<&SketchPrismTarget> {
        self.boolean_target()
            .filter(|target| target.operation() == PrismBoolean::Cut)
    }
    /// Read the target for a Boolean. Example: `definition.boolean_target()` before ordered replay.
    pub fn boolean_target(&self) -> Option<&SketchPrismTarget> {
        self.boolean_target.as_ref()
    }
}
