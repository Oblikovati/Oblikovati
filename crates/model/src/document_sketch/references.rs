// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches};
use crate::{
    identity::DocumentGuid,
    sketch::{SketchEntityId, SketchError},
};
use std::collections::BTreeMap;
/// A durable name resolves to the current owning sketch and optional entity identity.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchReferenceTarget {
    pub sketch: DocumentSketchId,
    pub entity: Option<SketchEntityId>,
}
/// One immutable current owner catalogue; duplicate names decline before lookup.
pub struct DocumentSketchReferenceIndex {
    guid: DocumentGuid,
    targets: BTreeMap<String, SketchReferenceTarget>,
}
impl DocumentSketchReferenceIndex {
    /// Build and check the current document namespace once.
    /// Example: `let names = DocumentSketchReferenceIndex::new(&snapshot.sketches, guid)?;`.
    pub fn new(sketches: &DocumentSketches, guid: DocumentGuid) -> Result<Self, SketchError> {
        let mut index = Self {
            guid,
            targets: BTreeMap::new(),
        };
        for placed in sketches.sketches() {
            index.insert_sketch(placed)?;
        }
        Ok(index)
    }
    fn insert_sketch(&mut self, placed: &super::PlacedSketch) -> Result<(), SketchError> {
        let sketch = placed.id();
        self.insert(
            self.guid.sketch_key(sketch.0).to_string(),
            SketchReferenceTarget {
                sketch,
                entity: None,
            },
        )?;
        for entity in placed.geometry().entity_ids() {
            self.insert(
                self.guid.sketch_entity_key(entity.0).to_string(),
                SketchReferenceTarget {
                    sketch,
                    entity: Some(*entity),
                },
            )?;
        }
        Ok(())
    }
    /// Resolve exact canonical text; unknown and deleted names have no target.
    /// Example: `names.resolve(&stored_key)`.
    pub fn resolve(&self, key: &str) -> Option<SketchReferenceTarget> {
        self.targets.get(key).copied()
    }
    /// Return a sketch name only after current-owner round-trip verification.
    /// Example: `let key = names.sketch_key(sketch)?;`.
    pub fn sketch_key(&self, sketch: DocumentSketchId) -> Result<String, SketchError> {
        let key = self.guid.sketch_key(sketch.0).to_string();
        self.verified_key(
            key,
            SketchReferenceTarget {
                sketch,
                entity: None,
            },
        )
    }
    /// Verify both entity identity and current ownership before returning its name.
    /// Example: `let key = names.entity_key(sketch, point)?;`.
    pub fn entity_key(
        &self,
        sketch: DocumentSketchId,
        entity: SketchEntityId,
    ) -> Result<String, SketchError> {
        let key = self.guid.sketch_entity_key(entity.0).to_string();
        self.verified_key(
            key,
            SketchReferenceTarget {
                sketch,
                entity: Some(entity),
            },
        )
    }
    fn verified_key(
        &self,
        key: String,
        expected: SketchReferenceTarget,
    ) -> Result<String, SketchError> {
        if self.resolve(&key) != Some(expected) {
            return Err(SketchError::Definition(format!(
                "sketch reference {key:?} resolves to {:?}: expected current target {expected:?}",
                self.resolve(&key)
            )));
        }
        Ok(key)
    }
    fn insert(&mut self, key: String, target: SketchReferenceTarget) -> Result<(), SketchError> {
        if let Some(previous) = self.targets.get(&key) {
            return Err(SketchError::Definition(format!(
                "ambiguous sketch reference {key:?}: targets {previous:?} and {target:?}; expected exactly one owner"
            )));
        }
        self.targets.insert(key.clone(), target);
        self.verified_key(key, target).map(|_| ())
    }
}
#[cfg(test)]
#[path = "../../tests/document_sketch/references.rs"]
mod tests;
