// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentSnapshot, sketch::SketchError,
};

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Change one owned sketch's display state; example: `commands.set_sketch_visibility(id, false)?`.
    pub fn set_sketch_visibility(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
    ) -> Result<(), CommandError<SketchError>> {
        self.session.set_sketch_visibility(sketch, visible)
    }
    /// Validate the metadata reply before adoption; example: `commands.set_sketch_visibility_projected(id, false, |reply| encode(reply?))?`.
    pub fn set_sketch_visibility_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        visible: bool,
        project: impl FnOnce(Result<&DocumentSnapshot, SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .set_sketch_visibility_projected(sketch, visible, project)
    }
}
