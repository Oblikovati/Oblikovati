// SPDX-License-Identifier: GPL-2.0-only
//! Parameter-driven ordered rigid body motion inputs (#4097).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey,
    fillet::reference_identity,
    work_geometry::{DatumAngle, DatumLength},
};
use crate::parameter::ParameterCollection;
mod operation;
use operation::{parameters, resolve, validate};

/// A body motion retains its named source and ordered drivers.
#[derive(Clone, Debug, PartialEq)]
pub struct MoveBodyDefinition {
    body: Vec<u8>,
    body_source: FeatureId,
    operations: Vec<MoveBodyOperation>,
}
/// One authored motion in document order.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum MoveBodyOperation {
    Translation([DatumLength; 3]),
    AlongRay {
        direction: [f64; 3],
        distance: DatumLength,
    },
    Rotation {
        origin: [DatumLength; 3],
        direction: [f64; 3],
        angle: DatumAngle,
    },
}
/// A finite rigid operation resolved for a kernel adapter.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum ResolvedBodyMotion {
    Translation([f64; 3]),
    Rotation {
        origin: [f64; 3],
        direction: [f64; 3],
        radians: f64,
    },
}
impl MoveBodyDefinition {
    /// Validate an ordered motion; example: `MoveBodyDefinition::new(body_key, operations)?`.
    pub fn new(body: Vec<u8>, operations: Vec<MoveBodyOperation>) -> Result<Self, String> {
        let (_, body_source) = reference_identity(&body, 4, "move body")?;
        if operations.is_empty() {
            return Err("move body operations []: expected at least one rigid operation".into());
        }
        for operation in &operations {
            validate(*operation)?;
        }
        Ok(Self {
            body,
            body_source,
            operations,
        })
    }
    /// Inspect the exact input name; example: `motion.body()` before replay.
    pub fn body(&self) -> &[u8] {
        &self.body
    }
    /// Inspect its source feature; example: `motion.body_source()` for the dependency graph.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }
    /// Borrow operations in their authored order; example: `motion.operations()` for editing.
    pub fn operations(&self) -> &[MoveBodyOperation] {
        &self.operations
    }
    /// Collect source and parameter reads; example: `motion.reads()` for invalidation.
    pub fn reads(&self) -> FeatureReadFootprint {
        let body = std::iter::once(FeatureReadKey::Feature(self.body_source));
        let parameters = self
            .operations
            .iter()
            .flat_map(|operation| parameters(*operation))
            .map(FeatureReadKey::Parameter);
        FeatureReadFootprint::new(body.chain(parameters), false)
    }
    /// Resolve current operation values; example: `motion.resolve(&parameters)?` before replay.
    pub fn resolve(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<Vec<ResolvedBodyMotion>, String> {
        self.operations
            .iter()
            .map(|operation| resolve(*operation, parameters))
            .collect()
    }
    /// Compare authored values exactly; example: `before.same_recipe(&after)` during history updates.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.body == other.body
            && self.operations.len() == other.operations.len()
            && self
                .operations
                .iter()
                .zip(&other.operations)
                .all(|(a, b)| operation::same(*a, *b))
    }
}
