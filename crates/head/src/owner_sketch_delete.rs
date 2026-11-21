// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn delete_gui_sketch_selection(&mut self) -> Result<bool, String> {
        self.synchronize_sketch_edit();
        if self.sketch_tool.is_some() {
            return Ok(false);
        }
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(false);
        };
        self.require_edit_document(edit.document)?;
        let report = self.delete_editor_picks_command(edit)?;
        if report.entities.is_empty() && report.constraints.is_empty() {
            return Ok(false);
        }
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: "Sketch selection deleted.".into(),
        });
        Ok(true)
    }
    fn delete_editor_picks_command(
        &mut self,
        edit: crate::HeadSketchEdit,
    ) -> Result<oblikovati_model::sketch::SketchEntityDeletion, String> {
        self.files
            .commands(edit.document)
            .map_err(|cause| cause.to_string())?
            .delete_editor_selection(Some(edit.sketch), false)
            .map_err(|cause| cause.to_string())
    }
}
