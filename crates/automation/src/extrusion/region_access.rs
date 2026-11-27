// SPDX-License-Identifier: GPL-2.0-only
use super::*;

/// Geometry capability used before committing an Extrude command.
pub trait ExtrudeRegionAccess {
    /// Resolve a current profile index to durable authored lineage; example: `backend.region(snapshot, sketch, 0)?`.
    fn region(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String>;
    /// Resolve a profile bound to current Part datums.
    /// Example: `backend.region_with_datums(snapshot, sketch, 0, datums)?`.
    fn region_with_datums(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
        index: usize,
        _datums: &WorkGeometryEvaluation,
    ) -> Result<SketchPrismSelection, String> {
        self.region(snapshot, sketch, index)
    }
    /// Resolve profile indices in request order. Native backends can share the arrangement.
    /// Example: `backend.regions(snapshot, sketch, &[0, 1], None)?`.
    fn regions(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
        indices: &[usize],
        datums: Option<&WorkGeometryEvaluation>,
    ) -> Result<Vec<SketchPrismSelection>, String> {
        indices
            .iter()
            .map(|&index| match datums {
                Some(datums) => self.region_with_datums(snapshot, sketch, index, datums),
                None => self.region(snapshot, sketch, index),
            })
            .collect()
    }
    /// Validate the solved candidate and return the current body count; example: `backend.validate(candidate, feature)?`.
    fn validate(
        &self,
        candidate: &DocumentSnapshot,
        document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<usize, String>;
    /// Report a policy only when this validator can certify retained replay.
    /// Example: `backend.replay_policy()` before acquiring a prefix.
    fn replay_policy(&self) -> Option<oblikovati_martensite::PrismResolutionPolicy> {
        None
    }
    /// Validate a candidate against its current retained prefix when supported.
    /// Example: `backend.capture_with_prefix(history, document, candidate, feature, prefix)?`.
    fn capture_with_prefix(
        &self,
        _history: &oblikovati_model::history::DocumentEditHistory,
        _document: DocumentGuid,
        _candidate: &DocumentSnapshot,
        _feature: FeatureId,
        _prefix: &ExtrudeReplayPrefix,
    ) -> Result<Option<ExtrudePrefixReplay>, String> {
        Ok(None)
    }
    /// Opt into projecting feature health from document-owned checkpoints under this policy.
    /// Example: `backend.checkpoint_health_policy()` before a suppression edit.
    fn checkpoint_health_policy(&self) -> Option<oblikovati_martensite::PrismResolutionPolicy> {
        None
    }
    /// Replay a feature and return its failure reason, or an empty string when healthy or suppressed.
    /// Example: `backend.feature_health(candidate, feature)?`.
    fn feature_health(
        &self,
        _candidate: &DocumentSnapshot,
        _document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<String, String> {
        Err(format!(
            "feature {feature:?}: expected a feature replay service"
        ))
    }
}
