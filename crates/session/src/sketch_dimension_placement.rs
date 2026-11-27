// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, TransactionObserver, sketch_recompute::SketchUpdateMode,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchError},
};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Move a dimension annotation through history, preserving numerical verification.
    /// Example: `session.move_sketch_dimension(sketch, dimension, [2.0,3.0])?;`.
    pub fn move_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        position: [f64; 2],
    ) -> Result<(), CommandError<SketchError>> {
        self.move_sketch_dimension_projected(sketch, dimension, position, |reply| reply)
    }
    /// Prepare a reply before adopting metadata; rejected replies discard placement edits.
    /// Example: `session.move_sketch_dimension_projected(sketch, dimension, point, |reply| encode(reply?))?;`.
    pub fn move_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        position: [f64; 2],
        project: impl FnOnce(Result<(), SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run_with_recompute(
            "Move sketch dimension",
            |snapshot, _| {
                project(
                    snapshot
                        .sketches
                        .set_dimension_text_point(sketch, dimension, position),
                )
            },
            SketchUpdateMode::Explicit,
        )
    }
}
