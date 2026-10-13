// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{
    HeadWindow,
    placement_input::{PlacementAction, SketchPlacementState},
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
        self.start_sketch_placement(
            document,
            sketch,
            crate::SketchPlacementTool::StraightSlot { width },
        )
    }
    /// Start one recipe tool on the selected sketch; example: `owner.start_sketch_placement(document,sketch,SketchPlacementTool::Rectangle)?`.
    pub fn start_sketch_placement(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        kind: crate::SketchPlacementTool,
    ) -> Result<(), String> {
        self.require_edit_document(document)?;
        let plane = self.resolve_edit(document, sketch)?.plane;
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let tool = SketchPlacementState::new(document, sketch, generation, plane, kind)?;
        self.select_placement_sketch(document, sketch)?;
        self.placement_notice(tool.prompt());
        self.placement = Some(tool);
        Ok(())
    }
    /// Inspect the exact current cursor recipe without allocating model IDs; example: `owner.placement_preview()`.
    pub fn placement_preview(&self) -> Option<&SketchPlacementRecipe> {
        self.placement
            .as_ref()
            .filter(|tool| self.placement_context_valid(tool))
            .and_then(|tool| tool.preview.as_ref())
    }
    /// Test whether the selected placement tool is waiting for points; example: `owner.placement_active()`.
    pub fn placement_active(&self) -> bool {
        self.placement
            .as_ref()
            .is_some_and(|tool| self.placement_context_valid(tool))
    }
    pub(super) fn process_placement_inputs(
        &mut self,
        inputs: &[ViewportEvent],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let mut changed = self.synchronize_sketch_edit();
        changed |= self.check_placement_context();
        for input in inputs {
            changed |= self.process_document_inputs(&[*input])?;
            changed |= self.check_placement_context();
            changed |= self.process_placement_event(*input, window);
        }
        Ok(changed)
    }
    fn check_placement_context(&mut self) -> bool {
        let Some(tool) = &self.placement else {
            return false;
        };
        if self.placement_context_valid(tool) {
            return false;
        }
        let name = tool.tool.name();
        self.placement = None;
        self.placement_notice(format!("{name} canceled because the document changed."));
        true
    }
    fn placement_context_valid(&self, tool: &SketchPlacementState) -> bool {
        self.active_sketch_edit()
            .is_some_and(|edit| (edit.document, edit.sketch) == (tool.document, tool.sketch))
            && self.require_edit_document(tool.document).is_ok()
            && self
                .files
                .workspace()
                .session(tool.document)
                .is_ok_and(|session| session.generation() == tool.generation)
    }
    fn process_placement_event(&mut self, event: ViewportEvent, window: &dyn HeadWindow) -> bool {
        let Some(mut tool) = self.placement.take() else {
            return false;
        };
        let action = tool.event(event);
        if matches!(action, PlacementAction::Cancel) {
            self.placement_notice(format!("{} canceled.", tool.tool.name()));
            return true;
        }
        let changed = match action {
            PlacementAction::Cursor { pixel, place } => {
                self.placement_cursor(&mut tool, window, pixel, place);
                true
            }
            _ => false,
        };
        self.placement = Some(tool);
        changed
    }
    fn placement_cursor(
        &mut self,
        tool: &mut SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
        place: bool,
    ) {
        let outcome = window
            .sketch_point(pixel, tool.plane)
            .and_then(|point| tool.cursor(point, place))
            .and_then(|recipe| self.commit_placement_cursor(tool, recipe));
        match outcome {
            Ok(()) => self.placement_notice(tool.prompt()),
            Err(cause) => {
                tool.abandon_press();
                self.command_notice = Some(DesktopCommandNotice::PlacementFailed { cause });
            }
        }
    }
    fn commit_placement_cursor(
        &mut self,
        tool: &mut SketchPlacementState,
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
    fn placement_notice(&mut self, message: String) {
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message });
    }
}
