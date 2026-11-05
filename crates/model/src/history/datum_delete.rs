// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, sketch_delete::consumers::DeletionIncidences};
use crate::feature::{FeatureId, FeatureReadKey};
use std::collections::BTreeSet;
mod error;
pub use error::DocumentDatumDeleteError;

/// Removed datum and owned scalars; reservation marks remain outside the snapshot.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DocumentDatumDeleteReport {
    pub feature: FeatureId,
    pub parameters: Vec<u64>,
}
impl DocumentSnapshot {
    /// Delete an unused datum and its owned scalars as one candidate edit.
    /// Example: `snapshot.delete_datum(feature)?` inside the session command owner.
    pub fn delete_datum(
        &mut self,
        id: FeatureId,
    ) -> Result<DocumentDatumDeleteReport, DocumentDatumDeleteError> {
        self.validate_scalar_ownership()
            .map_err(DocumentDatumDeleteError::Ownership)?;
        let report = deletion_report(self, id)?;
        require_unused(self, &report)?;
        let mut candidate = self.clone();
        apply_deletion(&mut candidate, &report)?;
        *self = candidate;
        Ok(report)
    }
    pub(super) fn validate_removed_datum_owners(
        &self,
        candidate: &Self,
    ) -> Result<(), DocumentDatumDeleteError> {
        let removed = removed_consumers(self, candidate);
        for feature in self.work_geometry.features() {
            if feature.owned_scalars().is_empty()
                || candidate.work_geometry.get(feature.id()).is_ok()
            {
                continue;
            }
            let report = deletion_report(self, feature.id())?;
            require_unused_except(self, &report, &removed)?;
        }
        Ok(())
    }
}
fn deletion_report(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<DocumentDatumDeleteReport, DocumentDatumDeleteError> {
    let feature = snapshot
        .work_geometry
        .get(id)
        .map_err(DocumentDatumDeleteError::Feature)?;
    Ok(DocumentDatumDeleteReport {
        feature: id,
        parameters: feature
            .owned_scalars()
            .iter()
            .map(|binding| binding.parameter)
            .collect(),
    })
}
fn deletion_keys(report: &DocumentDatumDeleteReport) -> BTreeSet<FeatureReadKey> {
    report
        .parameters
        .iter()
        .copied()
        .map(FeatureReadKey::Parameter)
        .chain([FeatureReadKey::Feature(report.feature)])
        .collect()
}
fn require_unused(
    snapshot: &DocumentSnapshot,
    report: &DocumentDatumDeleteReport,
) -> Result<(), DocumentDatumDeleteError> {
    let removed = deletion_keys(report);
    require_unused_except(snapshot, report, &removed)
}
fn require_unused_except(
    snapshot: &DocumentSnapshot,
    report: &DocumentDatumDeleteReport,
    removed: &BTreeSet<FeatureReadKey>,
) -> Result<(), DocumentDatumDeleteError> {
    let incidences = DeletionIncidences::from_snapshot(snapshot);
    for key in deletion_keys(report) {
        let mut readers = incidences.external_consumers(key, removed);
        readers.extend(parameter_consumers(snapshot, key, removed)?);
        readers.sort_unstable();
        readers.dedup();
        refuse_consumers(key, readers)?;
    }
    Ok(())
}
fn parameter_consumers(
    snapshot: &DocumentSnapshot,
    key: FeatureReadKey,
    removed: &BTreeSet<FeatureReadKey>,
) -> Result<Vec<FeatureReadKey>, DocumentDatumDeleteError> {
    let FeatureReadKey::Parameter(id) = key else {
        return Ok(Vec::new());
    };
    let readers = snapshot
        .parameters
        .authored_readers(id)
        .map_err(DocumentDatumDeleteError::Parameter)?;
    Ok(external_parameter_reads(readers, removed))
}
fn apply_deletion(
    snapshot: &mut DocumentSnapshot,
    report: &DocumentDatumDeleteReport,
) -> Result<(), DocumentDatumDeleteError> {
    snapshot
        .work_geometry
        .remove(report.feature)
        .map_err(DocumentDatumDeleteError::Feature)?;
    for id in &report.parameters {
        snapshot
            .parameters
            .remove_for_owner(*id)
            .map_err(DocumentDatumDeleteError::Parameter)?;
    }
    snapshot
        .validate_scalar_ownership()
        .map_err(DocumentDatumDeleteError::Ownership)
}

fn refuse_consumers(
    target: FeatureReadKey,
    readers: Vec<FeatureReadKey>,
) -> Result<(), DocumentDatumDeleteError> {
    if readers.is_empty() {
        return Ok(());
    }
    Err(DocumentDatumDeleteError::Consumed { target, readers })
}

fn external_parameter_reads(
    readers: Vec<u64>,
    removed: &BTreeSet<FeatureReadKey>,
) -> Vec<FeatureReadKey> {
    readers
        .into_iter()
        .map(FeatureReadKey::Parameter)
        .filter(|reader| !removed.contains(reader))
        .collect()
}

fn removed_consumers(
    before: &DocumentSnapshot,
    candidate: &DocumentSnapshot,
) -> BTreeSet<FeatureReadKey> {
    let mut removed = BTreeSet::new();
    for id in before.parameters.ids() {
        if candidate.parameters.get(*id).is_none() {
            removed.insert(FeatureReadKey::Parameter(*id));
        }
    }
    removed.extend(removed_feature_consumers(before, candidate));
    removed.extend(removed_sketch_consumers(before, candidate));
    removed
}
fn removed_feature_consumers(
    before: &DocumentSnapshot,
    candidate: &DocumentSnapshot,
) -> Vec<FeatureReadKey> {
    before
        .work_geometry
        .features()
        .iter()
        .filter(|feature| candidate.work_geometry.get(feature.id()).is_err())
        .map(|feature| FeatureReadKey::Feature(feature.id()))
        .collect()
}
fn removed_sketch_consumers(
    before: &DocumentSnapshot,
    candidate: &DocumentSnapshot,
) -> Vec<FeatureReadKey> {
    before
        .sketches
        .sketches()
        .iter()
        .filter(|sketch| candidate.sketches.get(sketch.id()).is_err())
        .map(|sketch| FeatureReadKey::Sketch(sketch.id().0))
        .collect()
}
