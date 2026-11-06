// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{SketchPlacementTool, placement_input::SketchPlacementState};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn begin_line_scope(
        &mut self,
        tool: &mut SketchPlacementState,
    ) -> Result<(), String> {
        if tool.tool != SketchPlacementTool::Line {
            return Ok(());
        }
        self.files
            .commands(tool.document)
            .map_err(|cause| cause.to_string())?
            .begin("Line")
            .map_err(|cause| cause.to_string())?;
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        tool.line_scope = session.transaction_identity();
        tool.generation = session.generation();
        Ok(())
    }
    pub(super) fn cancel_line_scope(&mut self, tool: &SketchPlacementState) -> Result<(), String> {
        if tool.line_scope.is_none() || !self.files.workspace().documents().contains(&tool.document)
        {
            return Ok(());
        }
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        // #3830: a caller may have ended the tool's group and opened a different one.
        if session.transaction_identity() != tool.line_scope {
            return Ok(());
        }
        self.files
            .commands(tool.document)
            .map_err(|cause| cause.to_string())?
            .abort()
            .map_err(|cause| cause.to_string())
    }
    pub(super) fn line_scope_context_valid(&self, tool: &SketchPlacementState) -> bool {
        self.active_sketch_edit()
            .is_some_and(|edit| (edit.document, edit.sketch) == (tool.document, tool.sketch))
            && tool.line_scope.is_some()
            && self
                .files
                .workspace()
                .session(tool.document)
                .is_ok_and(|session| {
                    session.transaction_identity() == tool.line_scope
                        && session.generation() == tool.generation
                })
    }
}
