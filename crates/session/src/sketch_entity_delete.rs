// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityDeletion, SketchEntityId, SketchError},
};
/// Complete retired geometry and relations, or a decline before adoption.
/// Example: `let reply: SketchEntityDeleteProjectionInput = Ok(Default::default());`.
pub type SketchEntityDeleteProjectionInput = Result<SketchEntityDeletion, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Delete selected entities and dimensions through recompute and undo history.
    /// Example: `session.delete_sketch_entities(sketch, &[line])?;`.
    pub fn delete_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
    ) -> Result<SketchEntityDeletion, CommandError<SketchError>> {
        self.delete_sketch_entities_projected(sketch, entities, |reply| reply)
    }
    /// Prepare the reply before adoption; rejected replies retain geometry and scalar ownership.
    /// Example: `session.delete_sketch_entities_projected(sketch, &[line], |reply| encode(reply?))?;`.
    pub fn delete_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        project: impl FnOnce(SketchEntityDeleteProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Delete sketch entities", |snapshot, _| {
            project(snapshot.delete_sketch_entities(sketch, entities))
        })
    }
}
