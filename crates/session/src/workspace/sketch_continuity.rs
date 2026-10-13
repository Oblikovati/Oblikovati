// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{SketchContinuityWatch, TransactionObserver};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchError};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Observe this document's sketch without a recipe edit; example: `commands.watch_sketch_continuity(sketch)?`.
    pub fn watch_sketch_continuity(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<SketchContinuityWatch, SketchError> {
        self.session.watch_sketch_continuity(sketch)
    }
}
