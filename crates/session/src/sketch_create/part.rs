// SPDX-License-Identifier: GPL-2.0-only
//! Create a sketch from generation-checked part datums.
use super::*;
use crate::{
    part_evaluation::{CurrentPartDatums, PartEvaluationError},
    sketch_recompute::SketchUpdateMode,
};
use oblikovati_model::{
    feature::work_geometry::WorkGeometryEvaluation,
    history::{DocumentSnapshot, RecipeReservations},
    identity::RecipeIdError,
    sketch::SketchError,
};
use std::{error::Error, fmt};

/// A current-part sketch creation was refused before adoption.
#[derive(Debug)]
pub enum PartSketchCreateError<E> {
    Evaluation(PartEvaluationError),
    Identity(RecipeIdError),
    Command(CommandError<E>),
}
impl<E: fmt::Display> fmt::Display for PartSketchCreateError<E> {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Evaluation(error) => error.fmt(output),
            Self::Identity(error) => error.fmt(output),
            Self::Command(error) => error.fmt(output),
        }
    }
}
impl<E: Error + 'static> Error for PartSketchCreateError<E> {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Evaluation(error) => Some(error),
            Self::Identity(error) => Some(error),
            Self::Command(error) => Some(error),
        }
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Create a sketch from current part data; example: `session.create_automatic_sketch_with_part_datums(&request, &datums)?`.
    pub fn create_automatic_sketch_with_part_datums(
        &mut self,
        request: &DocumentSketchCreation,
        datums: &CurrentPartDatums,
    ) -> Result<DocumentSketchId, PartSketchCreateError<DocumentSketchCreationError>> {
        self.create_automatic_sketch_with_part_datums_projected(request, datums, |result| result)
    }
    /// Prepare a reply before adoption; example: `session.create_automatic_sketch_with_part_datums_projected(&request, &datums, project)?`.
    pub fn create_automatic_sketch_with_part_datums_projected<R, E>(
        &mut self,
        request: &DocumentSketchCreation,
        datums: &CurrentPartDatums,
        project: impl FnOnce(SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, PartSketchCreateError<E>> {
        let (name, next) =
            crate::sketch_auto_names::next_name(self.next_sketch_name, &self.snapshot().sketches)
                .map_err(PartSketchCreateError::Identity)?;
        let mut named = request.clone();
        named.name = name;
        let mut created = false;
        let result = self.create_sketch_with_part_datums_projected(&named, datums, |input| {
            created = input.is_ok();
            project(input)
        });
        if created && result.is_ok() {
            self.next_sketch_name = next;
        }
        result
    }
    /// Create a named sketch with current part inputs and prepare a reply.
    /// Example: `session.create_sketch_with_part_datums_projected(&request, &datums, project)?`.
    pub fn create_sketch_with_part_datums_projected<R, E>(
        &mut self,
        request: &DocumentSketchCreation,
        datums: &CurrentPartDatums,
        project: impl FnOnce(SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, PartSketchCreateError<E>> {
        let current = datums
            .current(self)
            .map_err(PartSketchCreateError::Evaluation)?
            .clone();
        let options = self.sketch_options;
        self.run_with_recompute(
            "Create sketch",
            |snapshot, ids| project(create_and_solve(snapshot, ids, request, &current, options)),
            SketchUpdateMode::Explicit,
        )
        .map_err(PartSketchCreateError::Command)
    }
}
fn create_and_solve(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    request: &DocumentSketchCreation,
    datums: &WorkGeometryEvaluation,
    options: oblikovati_solve::nonlinear::SolveOptions,
) -> SketchCreationProjectionInput {
    let mut candidate = snapshot.clone();
    let sketch = candidate.create_document_sketch_with_datums(request, ids, datums)?;
    let solved = candidate.solve_pending_document_sketches_with_datums(options, || false, datums);
    let report = solved
        .report
        .reports
        .iter()
        .find(|report| report.sketch == sketch)
        .ok_or_else(|| {
            DocumentSketchCreationError::Sketch(SketchError::Definition(format!(
                "sketch {sketch:?} solve report: expected a current result"
            )))
        })?;
    report
        .result
        .as_ref()
        .map_err(|cause| DocumentSketchCreationError::Sketch(cause.clone()))?;
    candidate.sketches = solved.candidate;
    *snapshot = candidate;
    Ok(sketch)
}
