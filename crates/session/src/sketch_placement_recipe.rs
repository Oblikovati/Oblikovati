// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        SketchPlacementCreationError, SketchPlacementCreationReport, SketchPlacementFieldLock,
    },
    sketch::SketchPlacementRecipe,
};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Commit the preview's recipe and locked fields in one history command.
    /// Example: `session.create_sketch_placement(sketch,&recipe,&locked_fields)?;`.
    pub fn create_sketch_placement(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<SketchPlacementCreationReport, CommandError<SketchPlacementCreationError>> {
        let options = self.sketch_options;
        self.run("Create constrained sketch recipe", |snapshot, ids| {
            snapshot.create_sketch_placement(sketch, recipe, locks, options, ids)
        })
    }
}
