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
    pub(super) fn resolved_placement_preview(
        &mut self,
        tool: &crate::placement_input::SketchPlacementState,
    ) -> Result<Option<oblikovati_viewport::ViewportSketchPreview>, String> {
        if tool.tool == crate::SketchPlacementTool::Line {
            return tool
                .line_input()
                .map(|input| self.preview_sketch_line_chain(tool.document, tool.sketch, &input))
                .transpose();
        }
        if tool.tool == crate::SketchPlacementTool::Point {
            return self.resolved_point_preview(tool);
        }
        if !tool.fields_engaged() || !tool.at_finishing_stage() {
            return Ok(None);
        }
        // The preview shows what Enter would commit: the shape at its typed sizes.
        let seeded = tool.typed_seed_recipe(&self.sketch_construction)?;
        let Some(recipe) = seeded.as_ref().or(tool.preview.as_ref()) else {
            return Ok(None);
        };
        let datums = self.placement_part_datums(tool.document)?;
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        let preview = match datums.as_ref() {
            Some(datums) => session
                .preview_sketch_placement_with_datums(
                    tool.sketch,
                    recipe,
                    &tool.locks_for(recipe)?,
                    &self.sketch_construction,
                    datums,
                )
                .map_err(|cause| cause.to_string()),
            None => session
                .preview_sketch_placement(
                    tool.sketch,
                    recipe,
                    &tool.locks_for(recipe)?,
                    &self.sketch_construction,
                )
                .map_err(|cause| cause.to_string()),
        }
        .map_err(|cause| format!("placement preview {}: {cause}", tool.field_prompt()))?;
        Ok(Some(placement_preview_packet(tool.plane, preview)))
    }
    /// Commit the displayed recipe through the document owner; example: `owner.commit_sketch_placement(document,sketch,&recipe,&locks)?;`.
    pub fn commit_sketch_placement(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        recipe: &SketchPlacementRecipe,
        locks: &[Option<SketchPlacementFieldLock>],
    ) -> Result<SketchPlacementCreationReport, String> {
        let datums = self.placement_part_datums(document)?;
        let mut commands = self
            .files
            .commands(document)
            .map_err(|cause| format!("placement document {document}: {cause}"))?;
        let result = match datums.as_ref() {
            Some(datums) => commands
                .create_sketch_placement_with_datums(
                    sketch,
                    recipe,
                    locks,
                    &self.sketch_construction,
                    datums,
                )
                .map_err(|cause| cause.to_string()),
            None => commands
                .create_sketch_placement_with_construction(
                    sketch,
                    recipe,
                    locks,
                    &self.sketch_construction,
                )
                .map_err(|cause| cause.to_string()),
        };
        result.map_err(|cause| {
            format!(
                "placement document {document}, sketch {}: {cause}",
                sketch.0
            )
        })
    }
    fn placement_part_datums(
        &mut self,
        document: u64,
    ) -> Result<Option<oblikovati_session::part_evaluation::CurrentPartDatums>, String> {
        use oblikovati_automation::PartUpdateAccess;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if !oblikovati_automation::requires_part_replay(session.snapshot()) {
            return Ok(None);
        }
        self.part_scenes
            .current_part_datums(&mut self.files, document)
            .map(Some)
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
pub(super) fn placement_preview_packet(
    plane: oblikovati_model::sketch::SketchPlane,
    preview: oblikovati_session::SketchPlacementPreview,
) -> oblikovati_viewport::ViewportSketchPreview {
    oblikovati_viewport::ViewportSketchPreview::Resolved {
        plane,
        geometry: Box::new(preview.geometry),
        members: preview.members,
    }
}
