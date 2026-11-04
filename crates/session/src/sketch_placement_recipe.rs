// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        SketchPlacementCreationError, SketchPlacementCreationReport, SketchPlacementFieldLock,
    },
    sketch::{SketchCircleConstruction, SketchPlacementRecipe},
};
/// Solved private-copy geometry and only the new placement's members.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPlacementPreview {
    pub geometry: oblikovati_model::sketch::Sketch2D,
    pub members: Vec<oblikovati_model::sketch::SketchEntityId>,
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Solve the commit operation on a private history owner without reserving live IDs.
    /// Example: `session.preview_sketch_placement(sketch, recipe, locks, backend)?`.
    pub fn preview_sketch_placement(
        &self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        construction: &dyn SketchCircleConstruction,
    ) -> Result<SketchPlacementPreview, CommandError<SketchPlacementCreationError>> {
        let options = self.sketch_options;
        self.history
            .preview(|snapshot, ids| {
                crate::sketch_recompute::apply_recipe_edit(
                    snapshot,
                    ids,
                    |snapshot, ids| {
                        snapshot.create_sketch_placement(sketch, recipe, locks, options, ids)
                    },
                    Some(options),
                    |snapshot, report| {
                        collect_placement_preview(snapshot, report, recipe, construction)
                    },
                )
            })
            .map_err(CommandError::Edit)
    }
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
                verify_placement(snapshot, &report, recipe, construction)?;
                Ok(report)
            },
        )
    }
}
fn collect_placement_preview(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    report: SketchPlacementCreationReport,
    recipe: &SketchPlacementRecipe,
    construction: &dyn SketchCircleConstruction,
) -> Result<SketchPlacementPreview, SketchPlacementCreationError> {
    verify_placement(snapshot, &report, recipe, Some(construction))?;
    Ok(SketchPlacementPreview {
        geometry: snapshot.sketches.get(report.sketch)?.geometry().clone(),
        members: report.members,
    })
}
fn verify_placement(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    report: &SketchPlacementCreationReport,
    recipe: &SketchPlacementRecipe,
    construction: Option<&dyn SketchCircleConstruction>,
) -> Result<(), SketchPlacementCreationError> {
    let geometry = snapshot.sketches.get(report.sketch)?.geometry();
    recipe.verify_circular_seeds(geometry, &report.members, construction)?;
    Ok(())
}
