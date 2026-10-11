// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations, sketch_ids};
use crate::{feature::FeatureId, identity::RecipeIdError};

impl RecipeReservations<'_> {
    /// Reserve a permanent feature identity; example: `let id = ids.allocate_feature()?;`.
    pub fn allocate_feature(&mut self) -> Result<FeatureId, RecipeIdError> {
        self.feature_ids.allocate().map(FeatureId)
    }
}
impl DocumentEditHistory {
    /// Read the mark required by save and copy; example: `history.feature_high_water()`.
    pub fn feature_high_water(&self) -> u64 {
        self.feature_ids.high_water()
    }
    /// Restore reservations without rewinding deleted IDs; example: `history.with_feature_reservations(mark)?`.
    pub fn with_feature_reservations(mut self, mark: u64) -> Result<Self, RecipeIdError> {
        self.feature_ids =
            sketch_ids::restore_mark(mark, plane_ids(&self.current), self.feature_high_water())?;
        Ok(self)
    }
    pub(super) fn validate_feature_ids(
        &self,
        candidate: &DocumentSnapshot,
        before: u64,
    ) -> Result<(), HistoryError> {
        sketch_ids::require_reserved(
            plane_ids(candidate),
            plane_ids(&self.current),
            before,
            self.feature_high_water(),
            HistoryError::UnreservedFeatureId,
        )
    }
}
pub(super) fn plane_ids(snapshot: &DocumentSnapshot) -> Vec<u64> {
    snapshot
        .work_planes
        .features()
        .iter()
        .map(|feature| feature.id().0)
        .collect()
}
