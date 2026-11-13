// SPDX-License-Identifier: GPL-2.0-only
//! Reserve occurrence session IDs independently of undo history.
use super::{DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations, sketch_ids};
use crate::identity::{RecipeIdAllocator, RecipeIdError};

impl RecipeReservations<'_> {
    /// Reserve an occurrence ID even if the edit fails; example: `let id = ids.allocate_occurrence()?;`.
    pub fn allocate_occurrence(&mut self) -> Result<u64, RecipeIdError> {
        self.occurrence_ids.allocate()
    }
}
impl DocumentEditHistory {
    /// Inspect the session's occurrence mark; example: `history.occurrence_high_water()`.
    pub fn occurrence_high_water(&self) -> u64 {
        self.occurrence_ids.high_water()
    }
    pub(super) fn with_occurrence_reservations(mut self, mark: u64) -> Result<Self, RecipeIdError> {
        self.occurrence_ids = sketch_ids::restore_mark(
            mark,
            authored_occurrence_ids(&self.current),
            self.occurrence_high_water(),
        )?;
        Ok(self)
    }
    pub(super) fn validate_occurrence_ids(
        &self,
        candidate: &DocumentSnapshot,
        before: u64,
    ) -> Result<(), HistoryError> {
        let ids = authored_occurrence_ids(candidate);
        RecipeIdAllocator::restore(Some(self.occurrence_high_water()), &ids)?;
        sketch_ids::require_reserved(
            ids,
            authored_occurrence_ids(&self.current),
            before,
            self.occurrence_high_water(),
            HistoryError::UnreservedOccurrenceId,
        )
    }
}
pub(super) fn authored_occurrence_ids(snapshot: &DocumentSnapshot) -> Vec<u64> {
    snapshot
        .assembly
        .as_ref()
        .into_iter()
        .flatten()
        .map(|row| row.session_id)
        .collect()
}
