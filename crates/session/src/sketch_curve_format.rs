// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch_recompute::SketchUpdateMode;
use crate::{CommandError, DocumentSession, MutationError, TransactionObserver};
use crate::{editor_selection_scope::EditorSelectionBaseline, selection::SketchPickTarget};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    history::EditError,
    sketch::{SketchCurveFormat, SketchCurveFormatChange, SketchError},
};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Convert owned editor picks in one metadata history step, retaining checked identities.
    /// Example: `session.toggle_editor_curve_format(sketch, SketchCurveFormat::Centerline)?;`.
    pub fn toggle_editor_curve_format(
        &mut self,
        sketch: DocumentSketchId,
        format: SketchCurveFormat,
    ) -> Result<SketchCurveFormatChange, CommandError<SketchError>> {
        let (candidate, report) = self
            .prepare_editor_curve_format(sketch, format)
            .map_err(curve_format_error)?;
        // #3845: arming a mode and an unchanged axis role must not advance document generation.
        if report.changed.is_empty() {
            return Ok(report);
        }
        self.adopt_editor_curve_format(sketch, format, candidate, report)
    }
    fn adopt_editor_curve_format(
        &mut self,
        sketch: DocumentSketchId,
        format: SketchCurveFormat,
        candidate: DocumentSketches,
        report: SketchCurveFormatChange,
    ) -> Result<SketchCurveFormatChange, CommandError<SketchError>> {
        let before = EditorSelectionBaseline::capture(self.snapshot(), self.generation());
        let picks = self.editor_selection.clone();
        let result = self.run_with_recompute(
            format.label(),
            |snapshot, _| {
                snapshot.sketches = candidate;
                Ok(report)
            },
            SketchUpdateMode::Explicit,
        );
        match &result {
            Ok(_) => self.retain_editor_pick_identities(&picks),
            Err(_) => self.restore_refused_editor_picks(&before, Some(sketch), &picks),
        }
        result
    }
    fn prepare_editor_curve_format(
        &self,
        sketch: DocumentSketchId,
        format: SketchCurveFormat,
    ) -> Result<(DocumentSketches, SketchCurveFormatChange), SketchError> {
        let entities = self.checked_format_pick_entities(sketch)?;
        let mut candidate = self.snapshot().sketches.clone();
        let report = candidate.toggle_curve_format(sketch, &entities, format)?;
        Ok((candidate, report))
    }
    fn checked_format_pick_entities(
        &self,
        sketch: DocumentSketchId,
    ) -> Result<Vec<oblikovati_model::sketch::SketchEntityId>, SketchError> {
        self.editor_selection
            .validate_scope(self.document, self.generation(), sketch)?;
        Ok(self
            .editor_selection
            .items()
            .iter()
            .filter_map(|pick| match pick.target() {
                SketchPickTarget::Entity(id) => Some(id),
                _ => None,
            })
            .collect())
    }
}
fn curve_format_error(cause: SketchError) -> CommandError<SketchError> {
    CommandError::Edit(EditError::Mutation(MutationError::Command(cause)))
}
