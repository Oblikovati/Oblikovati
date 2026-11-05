// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadWindow, placement_input::SketchPlacementState, sketch_tool::SketchToolState};
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_viewport::ViewportEvent;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn retire_sketch_tool(&mut self) -> Result<(), String> {
        let Some(SketchToolState::Geometric(tool)) = self.sketch_tool.take() else {
            return Ok(());
        };
        if !self
            .files
            .workspace()
            .documents()
            .contains(&tool.edit.document)
        {
            return Ok(());
        }
        self.clear_geometric_picks(tool.edit)
    }
    pub(super) fn placement_state(&self) -> Option<&SketchPlacementState> {
        self.sketch_tool
            .as_ref()
            .and_then(SketchToolState::placement)
    }
    pub(super) fn sketch_context_valid(
        &self,
        document: u64,
        sketch: DocumentSketchId,
        generation: u64,
    ) -> bool {
        self.active_sketch_edit()
            .is_some_and(|edit| (edit.document, edit.sketch) == (document, sketch))
            && self.require_edit_document(document).is_ok()
            && self
                .files
                .workspace()
                .session(document)
                .is_ok_and(|session| session.generation() == generation)
    }
    pub(super) fn check_sketch_tool_context(&mut self) -> bool {
        let Some(tool) = &self.sketch_tool else {
            return false;
        };
        let (document, sketch, generation) = tool.context();
        if self.sketch_context_valid(document, sketch, generation) {
            return false;
        }
        let name = tool.label();
        let outcome = self.retire_sketch_tool();
        self.command_notice = Some(match outcome {
            Ok(()) => DesktopCommandNotice::PlacementPrompt {
                message: format!("{name} canceled because the document changed."),
            },
            Err(cause) => DesktopCommandNotice::AuthoringFailed { cause },
        });
        true
    }
    pub(super) fn process_sketch_tool_event(
        &mut self,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        // #3816: one press belongs to one tool, including the press that completes it.
        match self.sketch_tool.take() {
            Some(SketchToolState::Placement(tool)) => {
                self.process_placement_event(tool, event, window)
            }
            Some(SketchToolState::Geometric(tool)) => {
                self.process_geometric_event(tool, event, window)
            }
            None => self.process_sketch_pick(event, window),
        }
    }
}
