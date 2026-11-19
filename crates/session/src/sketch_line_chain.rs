// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, MutationError, TransactionObserver,
    sketch_recompute::apply_recipe_edit,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        DocumentSnapshot, EditError, RecipeReservations, SketchLineChainCreation,
        SketchLineChainCreationReport, SketchLineCreationError, SketchTangentArcCreation,
        SketchTangentArcReport,
    },
    sketch::{Sketch2D, SketchEntityId, SketchInferenceCandidates, SketchInferencePolicy},
};
use oblikovati_solve::nonlinear::SolveOptions;
/// Private-copy geometry and authored members, with the same inference evidence as commit.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLineChainPreview {
    pub geometry: Sketch2D,
    pub members: Vec<SketchEntityId>,
    pub report: SketchLineChainCreationReport,
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Preview against current document geometry without events, history changes, or owner reservations.
    /// Example: `session.preview_inferred_sketch_line_chain(sketch, &input)?;`.
    pub fn preview_inferred_sketch_line_chain(
        &self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<SketchLineChainPreview, CommandError<SketchLineCreationError>> {
        self.preview_scoped_line_chain(sketch, input)
    }
    /// Preview using the session's sketch scope; example: `session.preview_scoped_line_chain(sketch, &input)?`.
    pub fn preview_scoped_line_chain(
        &self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<SketchLineChainPreview, CommandError<SketchLineCreationError>> {
        let scope = self.sketch_inference_scope(sketch).map_err(scope_error)?;
        self.preview_line_chain_with_scope(sketch, input, Some(&scope))
    }
    fn preview_line_chain_with_scope(
        &self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<SketchLineChainPreview, CommandError<SketchLineCreationError>> {
        let options = self.sketch_options;
        self.history
            .preview(|snapshot, ids| {
                apply_recipe_edit(
                    snapshot,
                    ids,
                    |snapshot, ids| author_chain(snapshot, ids, sketch, input, options, scope),
                    Some(options),
                    |snapshot, report| {
                        for line in &report.lines {
                            line.verify_inference_preserved(snapshot, options)?;
                        }
                        chain_preview(snapshot, report)
                    },
                )
            })
            .map_err(CommandError::Edit)
    }
    /// Commit a converged chain with the same authoring and recompute used by preview.
    /// Example: `session.create_inferred_sketch_line_chain(sketch, &input)?;`.
    pub fn create_inferred_sketch_line_chain(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<SketchLineChainCreationReport, CommandError<SketchLineCreationError>> {
        self.create_scoped_line_chain(sketch, input)
    }
    /// Commit with the same scope used for preview; example: `session.create_scoped_line_chain(sketch, &input)?`.
    pub fn create_scoped_line_chain(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<SketchLineChainCreationReport, CommandError<SketchLineCreationError>> {
        let scope = self.sketch_inference_scope(sketch).map_err(scope_error)?;
        self.create_line_chain_with_scope(sketch, input, Some(&scope))
    }
    fn create_line_chain_with_scope(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
        scope: Option<&SketchInferenceCandidates>,
    ) -> Result<SketchLineChainCreationReport, CommandError<SketchLineCreationError>> {
        let options = self.sketch_options;
        self.run_projected(
            "Create Line",
            |snapshot, ids| author_chain(snapshot, ids, sketch, input, options, scope),
            |snapshot, report| {
                chain_solution(snapshot, report.sketch)?;
                for line in &report.lines {
                    line.verify_inference_preserved(snapshot, options)?;
                }
                Ok(report)
            },
        )
    }
    /// Place one point as its own history step; example: `session.create_sketch_point(id, &at)?`.
    pub fn create_sketch_point(
        &mut self,
        sketch: DocumentSketchId,
        placement: &oblikovati_model::history::SketchPointPlacement,
    ) -> Result<SketchEntityId, CommandError<SketchLineCreationError>> {
        let options = self.sketch_options;
        self.run("Create Point", |snapshot, ids| {
            snapshot.create_sketch_point(sketch, placement, options, ids)
        })
    }
    /// Create one arc off a line's end as its own history step.
    /// Example: `session.create_tangent_arc(sketch, &input)?;`.
    pub fn create_tangent_arc(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchTangentArcCreation,
    ) -> Result<SketchTangentArcReport, CommandError<SketchLineCreationError>> {
        let options = self.sketch_options;
        self.run_projected(
            "Create Tangent Arc",
            |snapshot, ids| snapshot.create_tangent_arc(sketch, input, options, ids),
            |snapshot, report| {
                chain_solution(snapshot, report.sketch)?;
                Ok(report)
            },
        )
    }
}
fn author_chain(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    sketch: DocumentSketchId,
    input: &SketchLineChainCreation,
    options: SolveOptions,
    scope: Option<&SketchInferenceCandidates>,
) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
    let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
    let report = snapshot
        .create_inferred_sketch_line_chain_scoped(sketch, input, policy, options, ids, scope)?;
    if let Some(first) = report.lines.first() {
        first.prepare_inference(snapshot, options)?;
    }
    Ok(report)
}
fn scope_error(
    error: oblikovati_model::sketch::SketchError,
) -> CommandError<SketchLineCreationError> {
    CommandError::Edit(EditError::Mutation(MutationError::Command(
        SketchLineCreationError::Sketch(error),
    )))
}
fn chain_solution(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
) -> Result<&Sketch2D, SketchLineCreationError> {
    snapshot
        .require_converged_sketch(sketch)
        .map_err(SketchLineCreationError::Readiness)
}
fn chain_preview(
    snapshot: &DocumentSnapshot,
    report: SketchLineChainCreationReport,
) -> Result<SketchLineChainPreview, SketchLineCreationError> {
    let geometry = chain_solution(snapshot, report.sketch)?.clone();
    let authored: std::collections::BTreeSet<_> = report
        .points
        .iter()
        .copied()
        .chain(report.lines.iter().map(|line| line.primitive.entity))
        .collect();
    let members = geometry
        .entity_ids()
        .iter()
        .copied()
        .filter(|id| authored.contains(id))
        .collect();
    Ok(SketchLineChainPreview {
        geometry,
        members,
        report,
    })
}
