// SPDX-License-Identifier: GPL-2.0-only

use crate::sketch_recompute::SketchUpdateMode;
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Run one command, joining a caller's group or completing an automatic group.
    /// Example: `session.run("Edit width", |snapshot, _| snapshot.parameters.set_expression(1, "2 cm"))?;`.
    pub fn run<R, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run_with_recompute(label, command, SketchUpdateMode::Automatic)
    }
    pub(crate) fn run_with_recompute<R, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        mode: SketchUpdateMode,
    ) -> Result<R, CommandError<E>> {
        let automatic = !self.in_transaction();
        if automatic {
            self.begin(label).map_err(CommandError::Session)?;
        }
        let result = self.edit_with_recompute(command, mode);
        if automatic {
            // A rejected candidate is unchanged, so end drops its frame without notifications.
            self.end().map_err(CommandError::Session)?;
        }
        result
    }
}
