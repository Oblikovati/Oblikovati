// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchDeleteError, DocumentSnapshot};
use crate::{
    document_sketch::DocumentSketchId,
    feature::{FeatureReadFootprint, FeatureReadKey},
};
use std::collections::{BTreeMap, BTreeSet};

/// A derived incidence index shares authored read keys with replay; no geometry or cache is queried.
#[derive(Default)]
pub(super) struct DeletionIncidences {
    readers: BTreeMap<FeatureReadKey, BTreeSet<FeatureReadKey>>,
    targets: BTreeMap<FeatureReadKey, BTreeSet<FeatureReadKey>>,
    unattributed: BTreeSet<FeatureReadKey>,
}

#[cfg(test)]
#[path = "consumer_tests.rs"]
mod tests;
impl DeletionIncidences {
    pub(super) fn from_snapshot(snapshot: &DocumentSnapshot) -> Self {
        let mut incidences = Self::default();
        for feature in snapshot.work_planes.features() {
            incidences.record(FeatureReadKey::Feature(feature.id()), feature.reads());
        }
        for sketch in snapshot.sketches.sketches() {
            incidences.record(
                FeatureReadKey::Sketch(sketch.id().0),
                &sketch.read_footprint(),
            );
        }
        incidences
    }
    fn record(&mut self, reader: FeatureReadKey, footprint: &FeatureReadFootprint) {
        if footprint.has_unattributed_reads() {
            self.unattributed.insert(reader);
        }
        let targets = footprint.keys().clone();
        for target in &targets {
            self.readers.entry(*target).or_default().insert(reader);
        }
        self.targets.insert(reader, targets);
    }
    pub(super) fn require_unused(
        &self,
        sketch: DocumentSketchId,
    ) -> Result<(), DocumentSketchDeleteError> {
        if !self.unattributed.is_empty() {
            return Err(DocumentSketchDeleteError::Unattributed {
                sketch,
                readers: self.unattributed.iter().copied().collect(),
            });
        }
        let key = FeatureReadKey::Sketch(sketch.0);
        let readers = self.consumer_keys(key);
        if readers.is_empty() {
            return Ok(());
        }
        Err(DocumentSketchDeleteError::Consumed { sketch, readers })
    }
    pub(super) fn has_consumers(&self, key: FeatureReadKey) -> bool {
        self.readers
            .get(&key)
            .is_some_and(|readers| !readers.is_empty())
    }
    pub(super) fn remove_consumer(
        &mut self,
        reader: FeatureReadKey,
    ) -> Result<(), DocumentSketchDeleteError> {
        let targets = self
            .targets
            .remove(&reader)
            .ok_or(DocumentSketchDeleteError::MissingIncidence(reader))?;
        for target in targets {
            self.remove_edge(reader, target)?;
        }
        Ok(())
    }
    fn consumer_keys(&self, key: FeatureReadKey) -> Vec<FeatureReadKey> {
        self.readers
            .get(&key)
            .into_iter()
            .flatten()
            .copied()
            .collect()
    }
    fn remove_edge(
        &mut self,
        reader: FeatureReadKey,
        target: FeatureReadKey,
    ) -> Result<(), DocumentSketchDeleteError> {
        let readers = self
            .readers
            .get_mut(&target)
            .ok_or(DocumentSketchDeleteError::MissingIncidence(target))?;
        if !readers.remove(&reader) {
            return Err(DocumentSketchDeleteError::MissingIncidence(reader));
        }
        Ok(())
    }
}
