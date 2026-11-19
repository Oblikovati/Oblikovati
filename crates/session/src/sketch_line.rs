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
        let scope = self.sketch_inference_scope(sketch);
        let options = self.sketch_options;
        self.run_projected(
            "Create inferred sketch line",
            |snapshot, ids| {
                let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
                let result = match scope {
                    Ok(scope) => snapshot.create_inferred_line_scoped(
                        sketch,
                        input,
                        policy,
                        options,
                        ids,
                        Some(&scope),
                    ),
                    Err(error) => Err(SketchLineCreationError::Sketch(error)),
                };
                let result = result.and_then(|report| {
                    report.prepare_inference(snapshot, options)?;
                    Ok(report)
                });
                Ok::<_, E>(result)
            },
            |snapshot, result| {
                let checked = result.and_then(|report| {
                    report.verify_inference_preserved(snapshot, options)?;
                    Ok(report)
                });
                project(checked)
            },
        )
    }
}
