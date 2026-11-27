// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{SketchContinuityResolver, SketchContinuityWatch, TransactionObserver};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchError};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Install the resolver used to check every published sketch state.
    /// Example: `commands.set_sketch_continuity_resolver(Box::new(resolver))`.
    pub fn set_sketch_continuity_resolver(&mut self, resolver: Box<dyn SketchContinuityResolver>) {
        self.session.set_sketch_continuity_resolver(resolver);
    }
    /// Observe this document's sketch without a recipe edit; example: `commands.watch_sketch_continuity(sketch)?`.
    pub fn watch_sketch_continuity(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<SketchContinuityWatch, SketchError> {
        self.session.watch_sketch_continuity(sketch)
    }
}
