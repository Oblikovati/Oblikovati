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
        self.run_recomputed_projected(label, command, mode, |_, result| Ok(result))
    }
    /// Inspect the solved candidate and prepare a reply before adoption; a refusal retains reserved IDs.
    /// Example: `session.run_projected("Create", create, |snapshot, report| verify_and_encode(snapshot,report))?`.
    pub fn run_projected<R, T, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        project: impl FnOnce(&DocumentSnapshot, R) -> Result<T, E>,
    ) -> Result<T, CommandError<E>> {
        self.run_recomputed_projected(label, command, SketchUpdateMode::Automatic, project)
    }
    /// Validate a candidate against its unchanged source history before adoption.
    /// Example: `session.run_projected_with_view("Edit", edit, validate)?`.
    pub fn run_projected_with_view<R, T, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        project: impl FnOnce(
            &DocumentSnapshot,
            R,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, E>,
    ) -> Result<T, CommandError<E>> {
        self.run_grouped(label, |session| {
            session.edit_projected_with_view(command, project)
        })
    }
    /// Evaluate a command on a private solved candidate without reserving live IDs.
    /// Example: `session.preview_projected(edit, |candidate, value| inspect(candidate, value))?`.
    pub fn preview_projected<R, T, E>(
        &self,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        project: impl FnOnce(&DocumentSnapshot, R) -> Result<T, E>,
    ) -> Result<T, CommandError<E>> {
        self.history
            .preview(|snapshot, ids| {
                crate::sketch_recompute::apply_recipe_edit(
                    snapshot,
                    ids,
                    command,
                    Some(self.sketch_options),
                    project,
                )
            })
            .map_err(CommandError::Edit)
    }
    fn run_recomputed_projected<R, T, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        mode: SketchUpdateMode,
        project: impl FnOnce(&DocumentSnapshot, R) -> Result<T, E>,
    ) -> Result<T, CommandError<E>> {
        self.run_grouped(label, |session| {
            session.edit_projected(command, mode, project)
        })
    }
    fn run_grouped<T, E>(
        &mut self,
        label: &str,
        operation: impl FnOnce(&mut Self) -> Result<T, CommandError<E>>,
    ) -> Result<T, CommandError<E>> {
        let automatic = !self.in_transaction();
        if automatic {
            self.begin(label).map_err(CommandError::Session)?;
        }
        let result = operation(self);
        if automatic {
            // A rejected candidate is unchanged, so end drops its frame without notifications.
            self.end().map_err(CommandError::Session)?;
        }
        result
    }
}
