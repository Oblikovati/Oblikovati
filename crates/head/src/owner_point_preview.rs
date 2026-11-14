// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::placement_input::SketchPlacementState;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn refresh_point_role_preview(&mut self) -> Result<(), String> {
        let preview = match self.placement_state() {
            Some(tool) if tool.tool == crate::SketchPlacementTool::Point => {
                self.resolved_point_preview(tool)?
            }
            _ => return Ok(()),
        };
        if let Some(crate::sketch_tool::SketchToolState::Placement(tool)) =
            self.sketch_tool.as_mut()
        {
            tool.resolved_preview = preview;
        }
        Ok(())
    }
    pub(super) fn resolved_point_preview(
        &self,
        tool: &SketchPlacementState,
    ) -> Result<Option<oblikovati_viewport::ViewportSketchPreview>, String> {
        let Some(position) = tool.pointer_position() else {
            return Ok(None);
        };
        let placement = tool.point_placement(position, None);
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        let preview = session
            .preview_sketch_point(tool.sketch, &placement)
            .map_err(|cause| format!("point preview {position:?}: {cause}"))?;
        Ok(Some(super::placement::placement_preview_packet(
            tool.plane, preview,
        )))
    }
}
