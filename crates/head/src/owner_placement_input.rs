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
        self.retire_sketch_tool()?;
        self.placement_notice(tool.prompt());
        self.sketch_tool = Some(crate::sketch_tool::SketchToolState::Placement(Box::new(
            tool,
        )));
        Ok(())
    }
    /// Inspect the exact current cursor recipe without allocating model IDs; example: `owner.placement_preview()`.
    pub fn placement_preview(&self) -> Option<&SketchPlacementRecipe> {
        self.placement_state()
            .filter(|tool| self.placement_context_valid(tool))
            .and_then(|tool| tool.preview.as_ref())
    }
    /// Test whether the selected placement tool is waiting for points; example: `owner.placement_active()`.
    pub fn placement_active(&self) -> bool {
        self.placement_state()
            .is_some_and(|tool| self.placement_context_valid(tool))
    }
    pub(super) fn process_placement_inputs(
        &mut self,
        inputs: &[ViewportEvent],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let mut changed = self.synchronize_sketch_edit();
        changed |= self.check_sketch_tool_context();
        for input in inputs {
            changed |= self.process_document_inputs(&[*input])?;
            changed |= self.check_sketch_tool_context();
            changed |= self.process_sketch_tool_event(*input, window);
        }
        Ok(changed)
    }
    fn placement_context_valid(&self, tool: &SketchPlacementState) -> bool {
        self.sketch_context_valid(tool.document, tool.sketch, tool.generation)
    }
    pub(super) fn process_placement_event(
        &mut self,
        mut tool: Box<SketchPlacementState>,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        let action = tool.event(event);
        if matches!(action, PlacementAction::Cancel) {
            self.placement_notice(format!("{} canceled.", tool.tool.name()));
            return true;
        }
        let changed = !matches!(action, PlacementAction::Idle);
        let completed = self.apply_placement_action(&mut tool, window, action);
        if !completed {
            self.sketch_tool = Some(crate::sketch_tool::SketchToolState::Placement(tool));
        }
        changed
    }
    fn apply_placement_action(
        &mut self,
        tool: &mut SketchPlacementState,
        window: &dyn HeadWindow,
        action: PlacementAction,
    ) -> bool {
        match action {
            PlacementAction::FieldsChanged => {
                self.refresh_placement_fields(tool);
                false
            }
            PlacementAction::CommitFields => self.commit_placement_fields(tool),
            PlacementAction::Cursor { pixel, place } => {
                self.placement_cursor(tool, window, pixel, place)
            }
            _ => false,
        }
    }
    fn commit_placement_fields(&mut self, tool: &mut SketchPlacementState) -> bool {
        let recipe = tool.preview.clone();
        self.commit_placement_cursor(tool, recipe)
            .unwrap_or_else(|cause| {
                self.command_notice = Some(DesktopCommandNotice::PlacementFailed { cause });
                false
            })
    }
    fn refresh_placement_fields(&mut self, tool: &mut SketchPlacementState) {
        tool.resolved_preview = None;
        match self.resolved_placement_preview(tool) {
            Ok(preview) => {
                tool.resolved_preview = preview;
                self.placement_notice(tool.field_prompt());
            }
            Err(cause) => {
                self.command_notice = Some(DesktopCommandNotice::PlacementFailed { cause });
            }
        }
    }
    fn placement_cursor(
        &mut self,
        tool: &mut SketchPlacementState,
        window: &dyn HeadWindow,
        pixel: [i32; 2],
        place: bool,
    ) -> bool {
        let outcome = window
            .sketch_point(pixel, tool.plane)
            .and_then(|point| tool.cursor(point, place, &self.sketch_construction))
            .and_then(|recipe| {
                tool.resolved_preview = self.resolved_placement_preview(tool)?;
                self.commit_placement_cursor(tool, recipe)
            });
        match outcome {
            Ok(true) => {
                self.placement_notice(format!("{} placed.", tool.tool.name()));
                true
            }
            Ok(false) => {
                self.placement_notice(tool.field_prompt());
                false
            }
            Err(cause) => {
                tool.abandon_press();
                self.command_notice = Some(DesktopCommandNotice::PlacementFailed { cause });
                false
            }
        }
    }
    fn commit_placement_cursor(
        &mut self,
        tool: &mut SketchPlacementState,
        recipe: Option<SketchPlacementRecipe>,
    ) -> Result<bool, String> {
        let Some(recipe) = recipe else {
            return Ok(false);
        };
        let locks = tool.preview_locks()?;
        self.commit_sketch_placement(tool.document, tool.sketch, &recipe, &locks)?;
        if tool.tool == crate::SketchPlacementTool::Point {
            return Ok(true);
        }
        let generation = self
            .files
            .workspace()
            .session(tool.document)
            .map_err(|cause| cause.to_string())?
            .generation();
        tool.restart(generation);
        Ok(false)
    }
    fn placement_notice(&mut self, message: String) {
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message });
    }
}
