// SPDX-License-Identifier: GPL-2.0-only
//! Point previews use the same role authoring as commit without reserving live identities.
use crate::{CommandError, DocumentSession, SketchPlacementPreview, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentSnapshot, SketchLineCreationError, SketchPointPlacement},
    sketch::SketchEntityId,
};
impl<O: TransactionObserver> DocumentSession<O> {
    /// Author and solve a point on a private history copy; example: `session.preview_sketch_point(sketch, &placement)?`.
    pub fn preview_sketch_point(
        &self,
        sketch: DocumentSketchId,
        placement: &SketchPointPlacement,
    ) -> Result<SketchPlacementPreview, CommandError<SketchLineCreationError>> {
        let options = self.sketch_options;
        self.history
            .preview(|snapshot, ids| {
                crate::sketch_recompute::apply_recipe_edit(
                    snapshot,
                    ids,
                    |snapshot, ids| snapshot.create_sketch_point(sketch, placement, options, ids),
                    Some(options),
                    |snapshot, point| point_preview(snapshot, sketch, point),
                )
            })
            .map_err(CommandError::Edit)
    }
}
fn point_preview(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    point: SketchEntityId,
) -> Result<SketchPlacementPreview, SketchLineCreationError> {
    let geometry = snapshot
        .sketches
        .get(sketch)
        .map_err(SketchLineCreationError::Sketch)?
        .geometry()
        .clone();
    Ok(SketchPlacementPreview {
        geometry,
        members: vec![point],
    })
}
