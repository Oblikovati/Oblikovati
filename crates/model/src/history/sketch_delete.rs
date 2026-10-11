// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::DocumentSketchId,
    feature::{FeatureId, FeatureReadKey},
};
mod consumers;
mod error;
use consumers::DeletionIncidences;
pub use error::DocumentSketchDeleteError;

/// Deleted stable identities; permanent reservation marks remain with the owner.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DocumentSketchDeleteReport {
    pub sketch: DocumentSketchId,
    pub construction_datums: Vec<FeatureId>,
}
impl DocumentSnapshot {
    /// Delete an unused sketch and the construction chain it orphans atomically; example: `snapshot.delete_sketch(id)?`.
    pub fn delete_sketch(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<DocumentSketchDeleteReport, DocumentSketchDeleteError> {
        let (incidences, candidates) = prepare_deletion(self, sketch)?;
        let mut candidate = self.clone();
        let report = apply_deletion(&mut candidate, sketch, incidences, candidates)?;
        *self = candidate;
        Ok(report)
    }
}
fn construction_candidates(
    snapshot: &DocumentSnapshot,
    incidences: &DeletionIncidences,
) -> Vec<FeatureId> {
    // The validated feature order is topological; reverse order permits one cascade pass (#3735).
    snapshot
        .work_geometry
        .features()
        .iter()
        .rev()
        .filter(|datum| {
            datum.is_construction_datum()
                && incidences.has_consumers(FeatureReadKey::Feature(datum.id()))
        })
        .map(|datum| datum.id())
        .collect()
}
fn prune_construction(
    snapshot: &mut DocumentSnapshot,
    incidences: &mut DeletionIncidences,
    candidates: Vec<FeatureId>,
) -> Result<Vec<FeatureId>, DocumentSketchDeleteError> {
    let mut deleted = Vec::new();
    for id in candidates {
        let key = FeatureReadKey::Feature(id);
        if incidences.has_consumers(key) {
            continue;
        }
        prune_one(snapshot, incidences, id)?;
        deleted.push(id);
    }
    Ok(deleted)
}

fn prepare_deletion(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
) -> Result<(DeletionIncidences, Vec<FeatureId>), DocumentSketchDeleteError> {
    snapshot
        .sketches
        .get(sketch)
        .map_err(DocumentSketchDeleteError::Sketch)?;
    let incidences = DeletionIncidences::from_snapshot(snapshot);
    incidences.require_unused(sketch)?;
    let candidates = construction_candidates(snapshot, &incidences);
    Ok((incidences, candidates))
}
fn apply_deletion(
    snapshot: &mut DocumentSnapshot,
    sketch: DocumentSketchId,
    mut incidences: DeletionIncidences,
    candidates: Vec<FeatureId>,
) -> Result<DocumentSketchDeleteReport, DocumentSketchDeleteError> {
    let sketches = &mut snapshot.sketches;
    sketches
        .remove(sketch)
        .map_err(DocumentSketchDeleteError::Sketch)?;
    incidences.remove_consumer(FeatureReadKey::Sketch(sketch.0))?;
    let construction_datums = prune_construction(snapshot, &mut incidences, candidates)?;
    Ok(DocumentSketchDeleteReport {
        sketch,
        construction_datums,
    })
}
fn prune_one(
    snapshot: &mut DocumentSnapshot,
    incidences: &mut DeletionIncidences,
    id: FeatureId,
) -> Result<(), DocumentSketchDeleteError> {
    snapshot
        .work_geometry
        .remove(id)
        .map_err(DocumentSketchDeleteError::Datum)?;
    incidences.remove_consumer(FeatureReadKey::Feature(id))
}
