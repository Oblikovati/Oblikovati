// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, MutationError, TransactionObserver};
use oblikovati_model::document_sketch::DocumentSketchRecompute;
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};
use oblikovati_solve::nonlinear::SolveOptions;
use std::convert::Infallible;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Recompute within normal history, joining an open edit group; example: `session.recompute_sketches(options, || false)?`.
    pub fn recompute_sketches(
        &mut self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchRecompute, CommandError<Infallible>> {
        self.run_with_recompute(
            "Recompute sketches",
            |snapshot, _| {
                let outcome = snapshot
                    .sketches
                    .solve(&snapshot.parameters, options, cancelled);
                snapshot.sketches = outcome.candidate;
                Ok(outcome.report)
            },
            SketchUpdateMode::Explicit,
        )
    }
}

#[derive(Clone, Copy)]
pub(crate) enum SketchUpdateMode {
    Automatic,
    Explicit,
}
impl SketchUpdateMode {
    pub(crate) fn options(self, options: SolveOptions) -> Option<SolveOptions> {
        match self {
            Self::Automatic => Some(options),
            Self::Explicit => None,
        }
    }
}
pub(crate) fn update_pending_sketches(snapshot: &mut DocumentSnapshot, options: SolveOptions) {
    let outcome = snapshot
        .sketches
        .solve_pending(&snapshot.parameters, options, || false);
    snapshot.sketches = outcome.candidate;
}
pub(crate) fn apply_recipe_edit<R, E>(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    options: Option<SolveOptions>,
) -> Result<R, MutationError<E>> {
    let result = command(snapshot, ids).map_err(MutationError::Command)?;
    snapshot
        .properties
        .sync_exposed_parameters(&snapshot.parameters)
        .map_err(MutationError::Property)?;
    if let Some(options) = options {
        update_pending_sketches(snapshot, options);
    }
    Ok(result)
}
