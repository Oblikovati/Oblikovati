// SPDX-License-Identifier: GPL-2.0-only
//! Identify sketches and datums that need ordered named-part recompute.
use super::PartFeatureDefinition;
use crate::{
    document_sketch::SketchPlacement, feature::work_plane::WorkPlaneReference,
    history::DocumentSnapshot,
};

impl DocumentSnapshot {
    /// Identify inputs that need named-part replay before publication.
    /// Example: `snapshot.requires_named_part_replay()` before recompute.
    pub fn requires_named_part_replay(&self) -> bool {
        if self
            .sketches
            .sketches()
            .iter()
            .any(|sketch| matches!(sketch.placement(), SketchPlacement::Face(_)))
        {
            return true;
        }
        self.work_geometry.features().iter().any(|feature| {
            let base = match feature.definition() {
                PartFeatureDefinition::Offset { base, .. } => Some(base),
                PartFeatureDefinition::ParallelThroughPoint(source) => Some(&source.base),
                PartFeatureDefinition::AtAngleAboutAxis(source) => Some(&source.base),
                _ => None,
            };
            matches!(
                base,
                Some(WorkPlaneReference::Face(_) | WorkPlaneReference::GoPrismFace(_))
            )
        })
    }
}
