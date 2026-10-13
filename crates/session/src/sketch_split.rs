// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityEditError, SketchEntityId, SketchLineSplitReport},
};
/// A complete split or a typed geometry/identity failure; example: `let reply: SketchSplitProjectionInput = Ok(report);`.
pub type SketchSplitProjectionInput = Result<SketchLineSplitReport, SketchEntityEditError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Recompute and prepare the reply before publishing a split; rejected replies retain reservations.
    /// Example: `session.split_sketch_line_projected(sketch, &[line], [2.0, 0.0], |result| result)?;`.
    pub fn split_sketch_line_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        pick: [f64; 2],
        project: impl FnOnce(SketchSplitProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Split sketch line", |snapshot, reservations| {
            project(snapshot.split_sketch_line(sketch, entities, pick, reservations))
        })
    }
}
