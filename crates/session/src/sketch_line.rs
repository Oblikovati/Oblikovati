// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchLineCreation, SketchLineCreationError, SketchLineCreationReport},
    sketch::SketchInferencePolicy,
};
/// Creation evidence or a typed decline before command adoption.
/// Example: `let reply: SketchLineProjectionInput = Ok(report);`.
pub type SketchLineProjectionInput = Result<SketchLineCreationReport, SketchLineCreationError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Create fresh endpoints and inferred relations using this document's current preferences.
    /// Example: `session.create_inferred_sketch_line(sketch, &input)?;`.
    pub fn create_inferred_sketch_line(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
    ) -> Result<SketchLineCreationReport, CommandError<SketchLineCreationError>> {
        self.create_inferred_sketch_line_projected(sketch, input, |result| result)
    }
    /// Prepare a response before adoption; rejected responses retain IDs and discard geometry.
    /// Example: `session.create_inferred_sketch_line_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_inferred_sketch_line_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
        project: impl FnOnce(SketchLineProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        let options = self.sketch_options;
        self.run("Create inferred sketch line", |snapshot, ids| {
            let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
            project(snapshot.create_inferred_sketch_line(sketch, input, policy, options, ids))
        })
    }
}
