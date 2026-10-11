// SPDX-License-Identifier: GPL-2.0-only
use super::{
    AuthoredFeature, FeatureId, FeatureSequence, WorkGeometryDefinition, WorkGeometryError,
};
use crate::feature::FeatureReplayRange;

impl WorkGeometryDefinition {
    pub(crate) fn require_plane(self) -> Result<(), WorkGeometryError> {
        if matches!(
            self,
            Self::Fixed(_)
                | Self::Offset { .. }
                | Self::ParallelThroughPoint(_)
                | Self::ThroughThreePoints(_)
                | Self::LineThroughPoint(_)
                | Self::NormalToAxis(_)
                | Self::ThroughTwoAxes(_)
                | Self::AtAngleAboutAxis(_)
        ) {
            return Ok(());
        }
        Err(WorkGeometryError::NonPlane(Box::new(self)))
    }
}
impl AuthoredFeature<WorkGeometryDefinition> {
    /// Read the authored orientation independently of evaluated geometry; example: `plane.is_normal_flipped()`.
    pub fn is_normal_flipped(&self) -> bool {
        self.plane_normal_flipped
    }
    pub(crate) fn validate_plane_orientation(&self) -> Result<(), WorkGeometryError> {
        if self.plane_normal_flipped {
            return self.definition.require_plane();
        }
        Ok(())
    }
}
impl FeatureSequence<WorkGeometryDefinition> {
    /// Reverse a placed plane and invalidate its dependent tail; example: `datums.flip_plane_normal(id)?`.
    pub fn flip_plane_normal(
        &mut self,
        id: FeatureId,
    ) -> Result<FeatureReplayRange, WorkGeometryError> {
        let position = self
            .position(id)
            .map_err(|_| WorkGeometryError::Unavailable {
                feature: id,
                expected: "placed plane",
            })?;
        self.features[position].definition.require_plane()?;
        self.features[position].plane_normal_flipped =
            !self.features[position].plane_normal_flipped;
        Ok(self.replay_from(position))
    }
}
