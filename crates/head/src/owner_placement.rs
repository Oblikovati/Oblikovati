// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        SketchLineChainCreation, SketchLineChainCreationReport, SketchPlacementCreationReport,
        SketchPlacementFieldLock,
    },
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
    /// Prepare current-document inference without publishing geometry or reserving permanent IDs.
    /// Example: `window.set_sketch_preview(Some(&owner.preview_sketch_line_chain(document, sketch, &input)?))?;`.
    pub fn preview_sketch_line_chain(
        &self,
        document: u64,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<oblikovati_viewport::ViewportSketchPreview, String> {
        let plane = self.resolve_edit(document, sketch)?.plane;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let preview = session
            .preview_inferred_sketch_line_chain(sketch, input)
            .map_err(|cause| cause.to_string())?;
        Ok(oblikovati_viewport::ViewportSketchPreview::Resolved {
            plane,
            geometry: Box::new(preview.geometry),
            members: preview.members,
        })
    }
    /// Commit the same inferred chain operation used by the current-document preview.
    /// Example: `owner.commit_sketch_line_chain(document, sketch, &input)?;`.
    pub fn commit_sketch_line_chain(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        input: &SketchLineChainCreation,
    ) -> Result<SketchLineChainCreationReport, String> {
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .create_inferred_sketch_line_chain(sketch, input)
            .map_err(|cause| {
                format!(
                    "line-chain placement document {document}, sketch {}: {cause}",
                    sketch.0
                )
            })
    }
}
