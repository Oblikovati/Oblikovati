// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, MutationError, TransactionObserver};
use oblikovati_model::document_sketch::{
    DocumentSketchId, DocumentSketchRecompute, DocumentSketchSolveOutcome,
};
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};
use oblikovati_model::sketch::SketchError;
use oblikovati_solve::diagnostics::ConstraintStructure;
use oblikovati_solve::nonlinear::SolveOptions;
use std::convert::Infallible;

/// Borrow a selected report or inspect its lookup decline before adoption; example: `let reply: SketchSolveProjectionInput<'_> = Ok(&report);`.
pub type SketchSolveProjectionInput<'report> =
    Result<&'report DocumentSketchRecompute, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Solve one sketch with this owner's policy and history; example: `session.solve_sketch(id, || false)?`.
    pub fn solve_sketch(
        &mut self,
        sketch: DocumentSketchId,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchRecompute, CommandError<SketchError>> {
        self.solve_sketch_projected(sketch, cancelled, |report| report.cloned())
    }
    /// Prepare a response before candidate adoption; example: `session.solve_sketch_projected(id, || false, |report| report.cloned())?`.
    pub fn solve_sketch_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        cancelled: impl FnMut() -> bool,
        project: impl FnOnce(SketchSolveProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        let options = self.sketch_options;
        self.run_with_recompute(
            "Solve sketch",
            |snapshot, _| solve_selected(snapshot, sketch, options, cancelled, project),
            SketchUpdateMode::Explicit,
        )
    }

    /// Inspect entity geometry and movement under this owner's rank policy.
    /// Example: `session.inspect_sketch_entities(id)?;`.
    pub fn inspect_sketch_entities(
        &self,
        id: DocumentSketchId,
    ) -> Result<Vec<oblikovati_model::sketch::SketchEntityInspection>, SketchError> {
        self.snapshot().inspect_document_sketch(
            id,
            oblikovati_model::sketch::SketchMovementOptions {
                rank_tolerance: self.sketch_options.rank_tolerance,
                ..Default::default()
            },
        )
    }
    /// Inspect local structure with this owner's rank policy and no mutation; example: `session.analyze_sketch(id)?`.
    pub fn analyze_sketch(
        &self,
        sketch: DocumentSketchId,
    ) -> Result<ConstraintStructure, SketchError> {
        self.snapshot()
            .analyze_document_sketch(sketch, self.sketch_options.rank_tolerance)
    }
    /// Recompute within normal history, joining an open edit group; example: `session.recompute_sketches(options, || false)?`.
    pub fn recompute_sketches(
        &mut self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchRecompute, CommandError<Infallible>> {
        self.run_with_recompute(
            "Recompute sketches",
            |snapshot, _| {
                let outcome = snapshot.solve_document_sketches(options, cancelled);
                snapshot.sketches = outcome.candidate;
                Ok(outcome.report)
            },
            SketchUpdateMode::Explicit,
        )
    }
}

fn solve_selected<R, E>(
    snapshot: &mut DocumentSnapshot,
    sketch: DocumentSketchId,
    options: SolveOptions,
    cancelled: impl FnMut() -> bool,
    project: impl FnOnce(SketchSolveProjectionInput<'_>) -> Result<R, E>,
) -> Result<R, E> {
    let outcome = snapshot.solve_one_document_sketch(sketch, options, cancelled);
    project_selected(snapshot, outcome, project)
}

fn project_selected<R, E>(
    snapshot: &mut DocumentSnapshot,
    outcome: Result<DocumentSketchSolveOutcome, SketchError>,
    project: impl FnOnce(SketchSolveProjectionInput<'_>) -> Result<R, E>,
) -> Result<R, E> {
    let response = project(
        outcome
            .as_ref()
            .map(|solved| &solved.report)
            .map_err(Clone::clone),
    )?;
    if let Ok(outcome) = outcome {
        snapshot.sketches = outcome.candidate;
    }
    Ok(response)
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
    let outcome = snapshot.solve_pending_document_sketches(options, || false);
    snapshot.sketches = outcome.candidate;
}
pub(crate) fn apply_recipe_edit<R, T, E>(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    options: Option<SolveOptions>,
    project: impl FnOnce(&DocumentSnapshot, R) -> Result<T, E>,
) -> Result<T, MutationError<E>> {
    let result = command(snapshot, ids).map_err(MutationError::Command)?;
    snapshot
        .properties
        .sync_exposed_parameters(&snapshot.parameters)
        .map_err(MutationError::Property)?;
    if let Some(options) = options {
        update_pending_sketches(snapshot, options);
    }
    project(snapshot, result).map_err(MutationError::Command)
}
