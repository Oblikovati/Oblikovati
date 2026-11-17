// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentEditHistory, DocumentSnapshot, EditError, HistoryError, RecipeReservations};
use crate::identity::RecipeIdAllocator;
/// A snapshot-backed evaluator cannot allocate or publish recipe identities (#4080).
///
/// Example: `SnapshotReplay::new(&snapshot)?.history()` supplies an immutable replay history.
///
/// ```compile_fail
/// use oblikovati_model::history::{DocumentSnapshot, SnapshotReplay};
/// fn mutate(snapshot: &DocumentSnapshot) {
///     let replay = SnapshotReplay::new(snapshot).unwrap();
///     replay.history().begin("edit").unwrap();
/// }
/// ```
pub struct SnapshotReplay(DocumentEditHistory);

impl SnapshotReplay {
    /// Reserve live IDs for evaluation; example: `SnapshotReplay::new(&snapshot)?`.
    pub fn new(snapshot: &DocumentSnapshot) -> Result<Self, HistoryError> {
        let highest = snapshot
            .parameters
            .ids()
            .iter()
            .copied()
            .chain(super::model_state_ids::authored_ids(snapshot))
            .max()
            .unwrap_or(0);
        let ids = RecipeIdAllocator::restore(Some(highest), snapshot.parameters.ids())?;
        Ok(Self(DocumentEditHistory::new(snapshot.clone(), ids)?))
    }

    /// Borrow the immutable history for an evaluator; example: `replay.history().snapshot()`.
    pub fn history(&self) -> &DocumentEditHistory {
        &self.0
    }
}

impl DocumentEditHistory {
    /// Validate an edit on a private current-state copy without consuming owner IDs or history.
    /// Example: `history.preview(|snapshot, ids| snapshot.create_sketch_primitive(sketch, &input, ids))?;`.
    pub fn preview<R, E>(
        &self,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    ) -> Result<R, EditError<E>> {
        let mut private = self.preview_owner().map_err(EditError::History)?;
        let token = private
            .begin("Transient preview")
            .map_err(EditError::History)?;
        private.edit(&token, command)
    }
    fn preview_owner(&self) -> Result<Self, HistoryError> {
        let ids =
            RecipeIdAllocator::restore(Some(self.high_water()), self.current.parameters.ids())?;
        Self::new(self.current.clone(), ids)?
            .with_sketch_reservations(self.sketch_reservations())?
            .with_feature_reservations(self.feature_high_water())?
            .with_occurrence_reservations(self.occurrence_high_water())?
            .with_derived_reservations(self.derived_high_water())
            .map_err(HistoryError::from)
    }
}
