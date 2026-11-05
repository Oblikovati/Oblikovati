// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        SketchPlacementCreationError, SketchPlacementCreationReport, SketchPlacementFieldLock,
    },
    sketch::{SketchCircleConstruction, SketchPlacementRecipe},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Use the document owner to commit one placement; example: `commands.create_sketch_placement(sketch,&recipe,&locks)?;`.
    pub fn create_sketch_placement(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<SketchPlacementCreationReport, CommandError<SketchPlacementCreationError>> {
        self.session.create_sketch_placement(sketch, recipe, locks)
    }
    /// Commit with the owner's circular construction service; example: `commands.create_sketch_placement_with_construction(sketch,&recipe,&[],backend)?`.
    pub fn create_sketch_placement_with_construction(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        construction: &dyn SketchCircleConstruction,
    ) -> Result<SketchPlacementCreationReport, CommandError<SketchPlacementCreationError>> {
        self.session
            .create_sketch_placement_with_construction(sketch, recipe, locks, construction)
    }
}
