// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations};

impl RecipeReservations<'_> {
    /// Permanently reserve a signed table ID, independently of parameter IDs.
    /// Example: `let table = reservations.allocate_derived_table()?;` before creating its links.
    pub fn allocate_derived_table(&mut self) -> Result<i64, HistoryError> {
        let next = self
            .derived_high_water
            .checked_add(1)
            .ok_or(HistoryError::DerivedIdentityExhausted)?;
        *self.derived_high_water = next;
        Ok(next)
    }
}

impl DocumentEditHistory {
    pub(super) fn validate_derived_ids(
        &self,
        candidate: &DocumentSnapshot,
        reserved_before: i64,
    ) -> Result<(), HistoryError> {
        for table in candidate.parameters.derived_tables() {
            if self.current.parameters.derived_table(table.id).is_none()
                && (table.id <= reserved_before || table.id > self.derived_high_water)
            {
                return Err(HistoryError::UnreservedDerivedIdentity(table.id));
            }
        }
        Ok(())
    }
}
