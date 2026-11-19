// SPDX-License-Identifier: GPL-2.0-only
//! Attach live display changes to the same transaction as their part-view edit (#4118).
use crate::{DocumentSession, TransactionObserver};
use oblikovati_model::history::{
    DocumentEditHistory, DocumentSnapshot, EvaluationTransition, PartViewDisplayTransition,
    RecipeReservations,
};

impl<O: TransactionObserver> DocumentSession<O> {
    pub(super) fn run_part_view_edit_projected<T>(
        &mut self,
        label: &str,
        display: PartViewDisplayTransition,
        edit: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<(), String>,
        project: impl FnOnce(&DocumentSnapshot, &DocumentEditHistory) -> Result<T, String>,
    ) -> Result<(T, EvaluationTransition), String> {
        self.begin(label).map_err(|cause| cause.to_string())?;
        let result = self.edit_projected_with_transition(edit, |snapshot, (), history| {
            project(snapshot, history)
        });
        let result = match result {
            Ok(result) => result,
            Err(cause) => {
                self.abort()
                    .map_err(|abort| format!("{cause}; abort failed: {abort}"))?;
                return Err(cause.to_string());
            }
        };
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
        self.end().map_err(|cause| cause.to_string())?;
        Ok(result)
    }
}
