// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, TransactionObserver, sketch_recompute::SketchUpdateMode,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentSnapshot, sketch::SketchError,
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Change saved display state without solving; example: `session.set_sketch_visibility(id, false)?`.
    pub fn set_sketch_visibility(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
    ) -> Result<(), CommandError<SketchError>> {
        self.set_sketch_visibility_projected(sketch, visible, |reply| reply.map(|_| ()))
    }
    /// Prepare a reply before metadata adoption; example: `session.set_sketch_visibility_projected(id, false, |reply| encode(reply?))?`.
    pub fn set_sketch_visibility_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
        project: impl FnOnce(Result<&DocumentSnapshot, SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        // #3826: an unchanged display value must not retire a live tool by advancing generation.
        if self
            .snapshot()
            .sketches
            .get(sketch)
            .is_ok_and(|current| current.visible() == visible)
        {
            return project(Ok(self.snapshot())).map_err(|cause| {
                CommandError::Edit(oblikovati_model::history::EditError::Mutation(
                    crate::MutationError::Command(cause),
                ))
            });
        }
        self.run_with_recompute(
            "Set sketch visibility",
            |snapshot, _| {
                let result = snapshot.sketches.set_visible(sketch, visible);
                project(result.map(|_| &*snapshot))
            },
            SketchUpdateMode::Explicit,
        )
    }
}
