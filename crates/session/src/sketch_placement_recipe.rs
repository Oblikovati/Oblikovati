// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        SketchPlacementCreationError, SketchPlacementCreationReport, SketchPlacementFieldLock,
    },
    sketch::{SketchCircleConstruction, SketchPlacementRecipe},
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
        self.create_placement_checked(sketch, recipe, locks, None)
    }
    /// Verify constructed curves after automatic solve; example: `session.create_sketch_placement_with_construction(sketch,&recipe,&[],backend)?`.
    pub fn create_sketch_placement_with_construction(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        construction: &dyn SketchCircleConstruction,
    ) -> Result<SketchPlacementCreationReport, CommandError<SketchPlacementCreationError>> {
        self.create_placement_checked(sketch, recipe, locks, Some(construction))
    }
    fn create_placement_checked(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        construction: Option<&dyn SketchCircleConstruction>,
    ) -> Result<SketchPlacementCreationReport, CommandError<SketchPlacementCreationError>> {
        let options = self.sketch_options;
        self.run_projected(
            "Create constrained sketch recipe",
            |snapshot, ids| snapshot.create_sketch_placement(sketch, recipe, locks, options, ids),
            |snapshot, report| {
                let geometry = snapshot.sketches.get(sketch)?.geometry();
                recipe.verify_circular_seeds(geometry, &report.members, construction)?;
                Ok(report)
            },
        )
    }
}
