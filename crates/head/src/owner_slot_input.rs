// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    HeadWindow,
    placement_slot::{PlacementAction, SlotPlacement},
};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlacementRecipe};
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
    /// Activate slot placement on an explicitly selected sketch; example: `owner.start_slot_placement(document,sketch,0.5)?`.
    pub fn start_slot_placement(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        width: f64,
    ) -> Result<(), String> {
        self.require_slot_document(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let plane = session
            .snapshot()
            .resolve_sketch_plane(sketch)
            .map_err(|cause| cause.to_string())?;
        let tool = SlotPlacement::new(document, sketch, session.generation(), plane, width)?;
        self.slot_notice(tool.prompt());
        self.slot_placement = Some(tool);
        Ok(())
    }
    /// Inspect the exact current cursor recipe without allocating model IDs; example: `owner.slot_placement_preview()`.
    pub fn slot_placement_preview(&self) -> Option<&SketchPlacementRecipe> {
        self.slot_placement
            .as_ref()
            .filter(|tool| self.slot_context_valid(tool))
            .and_then(|tool| tool.preview.as_ref())
    }
    /// Test whether the selected slot tool is waiting for points; example: `owner.slot_placement_active()`.
    pub fn slot_placement_active(&self) -> bool {
        self.slot_placement
            .as_ref()
            .is_some_and(|tool| self.slot_context_valid(tool))
    }
    fn require_slot_document(&self, document: u64) -> Result<(), String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "slot document {document}: expected the active document {:?}",
                self.files.workspace().active_document()
            ));
        }
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "slot document {document}: expected no open transaction"
            ));
        }
        Ok(())
    }
    pub(super) fn process_slot_inputs(
        &mut self,
        inputs: &[ViewportEvent],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let mut changed = self.check_slot_context();
        for input in inputs {
            changed |= self.process_document_inputs(&[*input])?;
            changed |= self.check_slot_context();
            changed |= self.process_slot_event(*input, window);
        }
        Ok(changed)
    }
    fn check_slot_context(&mut self) -> bool {
        let Some(tool) = &self.slot_placement else {
            return false;
        };
        if self.slot_context_valid(tool) {
            return false;
        }
        self.slot_placement = None;
        self.slot_notice("Slot canceled because the document changed.".into());
        true
    }
    fn slot_context_valid(&self, tool: &SlotPlacement) -> bool {
        self.require_slot_document(tool.document).is_ok()
            && self
                .files
                .workspace()
                .session(tool.document)
                .is_ok_and(|session| session.generation() == tool.generation)
    }
    fn process_slot_event(&mut self, event: ViewportEvent, window: &dyn HeadWindow) -> bool {
        let Some(mut tool) = self.slot_placement.take() else {
            return false;
        };
        let action = tool.event(event);
        if matches!(action, PlacementAction::Cancel) {
            self.slot_notice("Slot canceled.".into());
            return true;
        }
        let changed = match action {
            PlacementAction::Cursor { pixel, place } => {
                self.slot_cursor(&mut tool, window, pixel, place);
                true
            }
            _ => false,
        };
        self.slot_placement = Some(tool);
        changed
    }
    fn slot_cursor(
        &mut self,
        tool: &mut SlotPlacement,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
        place: bool,
    ) {
        let outcome = window
            .sketch_point(pixel, tool.plane)
            .and_then(|point| tool.cursor(point, place))
            .and_then(|recipe| self.commit_slot_cursor(tool, recipe));
        match outcome {
            Ok(()) => self.slot_notice(tool.prompt()),
            Err(cause) => {
                tool.abandon_press();
                self.command_notice = Some(DesktopCommandNotice::PlacementFailed { cause });
            }
        }
    }
    fn commit_slot_cursor(
        &mut self,
        tool: &mut SlotPlacement,
        recipe: Option<SketchPlacementRecipe>,
    ) -> Result<(), String> {
        let Some(recipe) = recipe else {
            return Ok(());
        };
        self.commit_sketch_placement(tool.document, tool.sketch, &recipe, &[])?;
        let generation = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?
            .generation();
        tool.restart(generation);
        Ok(())
    }
    fn slot_notice(&mut self, message: String) {
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message });
    }
}
