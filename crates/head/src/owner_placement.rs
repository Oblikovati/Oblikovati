// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchPlacementCreationReport, SketchPlacementFieldLock},
    sketch::SketchPlacementRecipe,
};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Commit the displayed recipe through the document owner; example: `owner.commit_sketch_placement(document,sketch,&recipe,&locks)?;`.
    pub fn commit_sketch_placement(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<SketchPlacementCreationReport, String> {
        self.files
            .commands(document)
            .map_err(|cause| format!("placement document {document}: {cause}"))?
            .create_sketch_placement_with_construction(
                sketch,
                recipe,
                locks,
                &self.sketch_construction,
            )
            .map_err(|cause| {
                format!(
                    "placement document {document}, sketch {}: {cause}",
                    sketch.0
                )
            })
    }
}
