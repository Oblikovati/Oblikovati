// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        EditError, SketchDimensionCreation, SketchDimensionCreationError,
        SketchDimensionCreationReport,
    },
};
/// Creation evidence or a typed decline before the transaction adopts the candidate.
/// Example: `let reply: SketchDimensionCreationProjectionInput = Ok(report);`.
pub type SketchDimensionCreationProjectionInput =
    Result<SketchDimensionCreationReport, SketchDimensionCreationError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Evaluate a dimension privately before asking for a role; example: `session.preview_sketch_dimension(id, &input)?`.
    pub fn preview_sketch_dimension(
        &self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
    ) -> Result<SketchDimensionCreationReport, EditError<SketchDimensionCreationError>> {
        self.history.preview(|snapshot, ids| {
            snapshot.create_sketch_dimension(sketch, input, self.sketch_options, ids)
        })
    }

    /// Author and evaluate a dimension through the normal history boundary.
    /// Example: `session.create_sketch_dimension(sketch, &input)?;`.
    pub fn create_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
    ) -> Result<SketchDimensionCreationReport, CommandError<SketchDimensionCreationError>> {
        self.create_sketch_dimension_projected(sketch, input, |result| result)
    }
    /// Prepare the response before publication; rejected responses retain attempted IDs only.
    /// Example: `session.create_sketch_dimension_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        project: impl FnOnce(SketchDimensionCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        let options = self.sketch_options;
        self.run("Create sketch dimension", |snapshot, ids| {
            project(snapshot.create_sketch_dimension(sketch, input, options, ids))
        })
    }
}
