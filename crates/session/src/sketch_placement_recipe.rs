// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, MutationError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        EditError, SketchPlacementContext, SketchPlacementCreationError,
        SketchPlacementCreationReport, SketchPlacementFieldLock,
    },
    sketch::{SketchCircleConstruction, SketchInferenceCandidates, SketchPlacementRecipe},
};
/// Solved private-copy geometry and only the new placement's members.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchPlacementPreview {
    pub geometry: oblikovati_model::sketch::Sketch2D,
    pub members: Vec<oblikovati_model::sketch::SketchEntityId>,
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Preview a placement against a generation-checked part replay.
    /// Example: `session.preview_sketch_placement_with_datums(id, recipe, locks, backend, datums)?`.
    pub fn preview_sketch_placement_with_datums(
        &self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        construction: &dyn SketchCircleConstruction,
        datums: &crate::part_evaluation::CurrentPartDatums,
    ) -> Result<SketchPlacementPreview, crate::PartSketchCreateError<SketchPlacementCreationError>>
    {
        let scope = self
            .sketch_inference_scope(sketch)
            .map_err(|error| crate::PartSketchCreateError::Command(scope_error(error)))?;
        let current = datums
            .current(self)
            .map_err(crate::PartSketchCreateError::Evaluation)?;
        let options = self.sketch_options;
        self.history
            .preview(|snapshot, ids| {
                crate::sketch_recompute::apply_recipe_edit(
                    snapshot,
                    ids,
                    |snapshot, ids| {
                        create_with_datums(
                            snapshot,
                            ids,
                            sketch,
                            recipe,
                            locks,
                            options,
                            (current, &scope),
                        )
                    },
                    None,
                    |snapshot, report| {
                        collect_placement_preview(snapshot, report, recipe, construction)
                    },
                )
            })
            .map_err(|error| crate::PartSketchCreateError::Command(CommandError::Edit(error)))
    }
    /// Commit a placement against the same current part inputs as its preview.
    /// Example: `session.create_sketch_placement_with_datums(id, recipe, locks, backend, datums)?`.
    pub fn create_sketch_placement_with_datums(
        &mut self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        construction: &dyn SketchCircleConstruction,
        datums: &crate::part_evaluation::CurrentPartDatums,
    ) -> Result<
        SketchPlacementCreationReport,
        crate::PartSketchCreateError<SketchPlacementCreationError>,
    > {
        let scope = self
            .sketch_inference_scope(sketch)
            .map_err(|error| crate::PartSketchCreateError::Command(scope_error(error)))?;
        let current = datums
            .current(self)
            .map_err(crate::PartSketchCreateError::Evaluation)?
            .clone();
        let options = self.sketch_options;
        self.run_with_recompute(
            "Create constrained sketch recipe",
            |snapshot, ids| {
                let report = create_with_datums(
                    snapshot,
                    ids,
                    sketch,
                    recipe,
                    locks,
                    options,
                    (&current, &scope),
                )?;
                verify_placement(snapshot, &report, recipe, Some(construction))?;
                Ok(report)
            },
            crate::sketch_recompute::SketchUpdateMode::Explicit,
        )
        .map_err(crate::PartSketchCreateError::Command)
    }
    /// Solve the commit operation on a private history owner without reserving live IDs.
    /// Example: `session.preview_sketch_placement(sketch, recipe, locks, backend)?`.
    pub fn preview_sketch_placement(
        &self,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
        construction: &dyn SketchCircleConstruction,
    ) -> Result<SketchPlacementPreview, CommandError<SketchPlacementCreationError>> {
        let scope = self.sketch_inference_scope(sketch).map_err(scope_error)?;
        let options = self.sketch_options;
        self.history
            .preview(|snapshot, ids| {
                crate::sketch_recompute::apply_recipe_edit(
                    snapshot,
                    ids,
                    |snapshot, ids| {
                        snapshot.create_scoped_sketch_placement(
                            sketch,
                            recipe,
                            locks,
                            options,
                            ids,
                            SketchPlacementContext {
                                datums: None,
                                scope: Some(&scope),
                            },
                        )
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
        let scope = self.sketch_inference_scope(sketch).map_err(scope_error)?;
        let options = self.sketch_options;
        self.run_projected(
            "Create constrained sketch recipe",
            |snapshot, ids| {
                snapshot.create_scoped_sketch_placement(
                    sketch,
                    recipe,
                    locks,
                    options,
                    ids,
                    SketchPlacementContext {
                        datums: None,
                        scope: Some(&scope),
                    },
                )
            },
            |snapshot, report| {
                verify_placement(snapshot, &report, recipe, construction)?;
                Ok(report)
            },
        )
    }
}
fn create_with_datums(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
    sketch: DocumentSketchId,
    recipe: &SketchPlacementRecipe,
    locks: &[Option<SketchPlacementFieldLock>],
    options: oblikovati_solve::nonlinear::SolveOptions,
    context: (
        &oblikovati_model::feature::work_geometry::WorkGeometryEvaluation,
        &SketchInferenceCandidates,
    ),
) -> Result<SketchPlacementCreationReport, SketchPlacementCreationError> {
    let (datums, scope) = context;
    let report = snapshot.create_scoped_sketch_placement(
        sketch,
        recipe,
        locks,
        options,
        ids,
        SketchPlacementContext {
            datums: Some(datums),
            scope: Some(scope),
        },
    )?;
    let solved =
        snapshot.solve_one_document_sketch_with_datums(sketch, options, || false, datums)?;
    let result = solved
        .report
        .reports
        .iter()
        .find(|entry| entry.sketch == sketch)
        .ok_or_else(|| {
            oblikovati_model::sketch::SketchError::Definition(format!(
                "sketch {sketch:?} placement solve: expected a current report"
            ))
        })?;
    result.result.as_ref().map_err(Clone::clone)?;
    snapshot.sketches = solved.candidate;
    Ok(report)
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

fn scope_error(
    error: oblikovati_model::sketch::SketchError,
) -> CommandError<SketchPlacementCreationError> {
    CommandError::Edit(EditError::Mutation(MutationError::Command(
        SketchPlacementCreationError::Sketch(error),
    )))
}
