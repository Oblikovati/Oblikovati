// SPDX-License-Identifier: GPL-2.0-only
//! Attach live display changes to the same transaction as their part-view edit (#4118).
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::history::{DocumentSnapshot, PartViewDisplayTransition, RecipeReservations};

impl<O: TransactionObserver> DocumentSession<O> {
    pub(super) fn run_part_view_edit(
        &mut self,
        label: &str,
        display: PartViewDisplayTransition,
        edit: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<(), String>,
    ) -> Result<(), String> {
        self.begin(label).map_err(|cause| cause.to_string())?;
        if let Err(cause) = self.edit(edit) {
            self.abort()
                .map_err(|abort| format!("{cause}; abort failed: {abort}"))?;
            return Err(cause.to_string());
        }
        let token = &self
            .group
            .as_ref()
            .ok_or("part view edit: expected open group")?
            .token;
        if let Err(cause) = self.history.tag_part_view_display(token, display) {
            self.abort()
                .map_err(|abort| format!("{cause}; abort failed: {abort}"))?;
            return Err(cause.to_string());
        }
        self.end().map_err(|cause| cause.to_string())
    }
}
