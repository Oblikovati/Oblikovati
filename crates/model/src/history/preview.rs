// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentEditHistory, DocumentSnapshot, EditError, HistoryError, RecipeReservations};
use crate::identity::RecipeIdAllocator;
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
            .with_derived_reservations(self.derived_high_water())
            .map_err(HistoryError::from)
    }
}
