// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, TransactionObserver, sketch_recompute::apply_recipe_edit,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        DocumentSnapshot, RecipeReservations, SketchLineChainCreation,
        SketchLineChainCreationReport, SketchLineCreationError,
    },
    sketch::{Sketch2D, SketchEntityId, SketchInferencePolicy},
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
        let options = self.sketch_options;
        self.history
            .preview(|snapshot, ids| {
                apply_recipe_edit(
                    snapshot,
                    ids,
                    |snapshot, ids| author_chain(snapshot, ids, sketch, input, options),
                    Some(options),
                    chain_preview,
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
        let options = self.sketch_options;
        self.run_projected(
            "Create inferred sketch line chain",
            |snapshot, ids| author_chain(snapshot, ids, sketch, input, options),
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
) -> Result<SketchLineChainCreationReport, SketchLineCreationError> {
    let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
    snapshot.create_inferred_sketch_line_chain(sketch, input, policy, options, ids)
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
