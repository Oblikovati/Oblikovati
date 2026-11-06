// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::placement_input::SketchPlacementState;
use oblikovati_model::document_sketch::DocumentSketches;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn finish_line_placement(
        &mut self,
        tool: &mut SketchPlacementState,
        closed: bool,
    ) -> Result<bool, String> {
        if !self.line_scope_context_valid(tool) {
            return Err(format!(
                "Line scope {:?}: expected the current owned transaction and sketch generation",
                tool.line_scope
            ));
        }
        let Some(input) = tool.line_input(false, closed) else {
            self.cancel_line_scope(tool)?;
            self.line_placement_notice("Line canceled.");
            return Ok(true);
        };
        // #3830: placement rejects zero-length edges; the typed line API deliberately admits them.
        oblikovati_model::sketch::SketchPlacementRecipe::line_chain(&input.points, input.closed)
            .map_err(|cause| cause.to_string())?;
        let before = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .sketches
            .clone();
        let result = self
            .commit_sketch_line_chain(tool.document, tool.sketch, &input)
            .and_then(|_| {
                self.files
                    .commands(tool.document)
                    .map_err(|cause| cause.to_string())?
                    .end()
                    .map_err(|cause| cause.to_string())
            });
        if let Err(cause) = result {
            self.restore_line_scope(tool, &before)?;
            return Err(cause);
        }
        self.line_placement_notice("Line chain placed.");
        Ok(true)
    }
    fn restore_line_scope(
        &mut self,
        tool: &mut SketchPlacementState,
        before: &DocumentSketches,
    ) -> Result<(), String> {
        self.cancel_line_scope(tool)?;
        let session = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() || !session.snapshot().sketches.same_recipe(before) {
            return Err(format!(
                "Line rollback document {}: expected the original sketch recipes and no caller transaction",
                tool.document
            ));
        }
        self.begin_line_scope(tool)
    }
    fn line_placement_notice(&mut self, message: &str) {
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: message.into(),
        });
    }
}
