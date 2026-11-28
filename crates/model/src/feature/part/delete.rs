// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    feature::{FeatureId, FeatureReadFootprint, FeatureReadKey},
    identity::{ReferenceEnvelope, topology_reference_document},
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
        topology_reference_document(&key)?;
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

impl<Body, Failure> super::PartFeatureEvaluation<Body, Failure> {
    pub(super) fn delete_body<B: super::PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &B,
        feature: FeatureId,
        definition: &DeleteBodyDefinition,
    ) -> Result<(), super::PartFeatureFailure<Failure>> {
        let source = definition.source();
        let slot = super::body_resolution::resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            source,
            definition.key(),
        )
        .map_err(super::PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.body_lineage.retire(slot);
        self.deleted.insert(feature, definition.key().to_vec());
        Ok(())
    }
}

impl crate::history::DocumentSnapshot {
    pub(super) fn part_replay_range(
        &self,
    ) -> Result<crate::feature::FeatureReplayRange, super::PartFeatureProgramError> {
        self.validate_prism_reads()?;
        self.work_geometry
            .validate_work_plane_reads()
            .map_err(super::PartFeatureProgramError::Reads)?;
        Ok(crate::feature::FeatureReplayRange {
            start: 0,
            end: self.work_geometry.effective_end(),
        })
    }
}

pub(super) fn require_replay_current(
    cancelled: bool,
    completed: usize,
) -> Result<(), super::PartFeatureProgramError> {
    if cancelled {
        return Err(super::PartFeatureProgramError::Cancelled { completed });
    }
    Ok(())
}
