// SPDX-License-Identifier: GPL-2.0-only
use crate::selection::{SelectionSet, SketchPick, SketchPickTarget};
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityDeletion, SketchError},
};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Mint a generation-bound handle only after checking its owner and relation category.
    /// Example: `session.sketch_pick(sketch, SketchPickTarget::Entity(circle))?;`.
    pub fn sketch_pick(
        &self,
        sketch: DocumentSketchId,
        target: SketchPickTarget,
    ) -> Result<SketchPick, SketchError> {
        let geometry = self.snapshot().sketches.get(sketch)?.geometry();
        match target {
            SketchPickTarget::Entity(id) => {
                geometry.entity_kind(id)?;
            }
            SketchPickTarget::Dimension(id) => {
                geometry.dimension_source(id)?;
            }
            SketchPickTarget::GeometricConstraint(id) => {
                geometry.geometric_constraint_index(id)?;
            }
        }
        Ok(SketchPick::checked(
            self.document,
            self.generation(),
            sketch,
            target,
        ))
    }
    /// Apply the editor Delete action once; no editor, active tool, or empty picks is a no-op.
    /// Clear picks only after adoption; failed edits and observer vetoes retain them (#1232/#2017).
    /// Example: `session.delete_selected_sketch_picks(Some(sketch), false, &mut picks)?;`.
    pub fn delete_selected_sketch_picks(
        &mut self,
        active_sketch: Option<DocumentSketchId>,
        tool_active: bool,
        picks: &mut SelectionSet<SketchPick>,
    ) -> Result<SketchEntityDeletion, CommandError<SketchError>> {
        let Some(sketch) = active_sketch.filter(|_| !tool_active && !picks.items().is_empty())
        else {
            return Ok(SketchEntityDeletion::default());
        };
        let (document, generation) = (self.document, self.generation());
        let report = self.run(picks.deletion_label(), |snapshot, _| {
            let intent = picks.deletion_intent(document, generation, sketch)?;
            snapshot.delete_sketch_selection(sketch, &intent)
        })?;
        picks.clear();
        Ok(report)
    }
}
