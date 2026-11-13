// SPDX-License-Identifier: GPL-2.0-only
//! Canvas selection, solved preview, and typed commit for the sketch Move command.
use super::*;
use crate::{HeadSketchEdit, HeadWindow, sketch_tool::SketchToolState};
use oblikovati_api::{client::Client, wire::TransformSketchArgs};
use oblikovati_gui::{GuiSketchMoveAction, GuiSketchMoveStage};
use oblikovati_model::sketch::{SketchEntityId, SketchSelectionTransform};
use oblikovati_session::selection::SelectionClickMode;
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportSketchPreview,
};

#[derive(Clone, Copy)]
enum MoveStage {
    Geometry,
    Base(SketchEntityId),
    Target(SketchEntityId, [f64; 2]),
}
pub(crate) struct SketchMoveState {
    pub edit: HeadSketchEdit,
    pub generation: u64,
    stage: MoveStage,
    pub preview: Option<ViewportSketchPreview>,
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn sketch_move_command(
        &mut self,
        action: GuiSketchMoveAction,
    ) -> Result<bool, String> {
        match action {
            GuiSketchMoveAction::Start => self.start_sketch_move(),
            GuiSketchMoveAction::Done => {
                if !matches!(self.sketch_tool, Some(SketchToolState::Move(_))) {
                    return Ok(false);
                }
                self.retire_sketch_tool()?;
                self.placement_notice("Ready".into());
                Ok(true)
            }
        }
    }
    fn start_sketch_move(&mut self) -> Result<bool, String> {
        let (edit, generation) = self.capture_sketch_tool_context()?;
        self.require_move_transaction_clear(edit.document)?;
        self.retire_sketch_tool()?;
        self.clear_geometric_picks(edit)?;
        self.sketch_tool = Some(SketchToolState::Move(Box::new(SketchMoveState {
            edit,
            generation,
            stage: MoveStage::Geometry,
            preview: None,
        })));
        self.sketch_memory.repeat_command = Some(oblikovati_gui::GuiSketchRepeat::Move);
        self.placement_notice("Move: Select geometry".into());
        Ok(true)
    }
    fn require_move_transaction_clear(&self, document: u64) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|e| e.to_string())?;
        if !session.in_transaction() {
            return Ok(());
        }
        Err(format!(
            "sketch move in document {document}: expected no open transaction"
        ))
    }
    pub(super) fn gui_sketch_move_stage(&self) -> Option<GuiSketchMoveStage> {
        let Some(SketchToolState::Move(tool)) = &self.sketch_tool else {
            return None;
        };
        Some(match tool.stage {
            MoveStage::Geometry => GuiSketchMoveStage::Geometry,
            MoveStage::Base(_) => GuiSketchMoveStage::Base,
            MoveStage::Target(..) => GuiSketchMoveStage::Target,
        })
    }
    pub(super) fn process_sketch_move(
        &mut self,
        mut tool: Box<SketchMoveState>,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        if is_cancel(event) {
            return self.cancel_sketch_move(tool.edit);
        }
        let outcome = self.move_event(&mut tool, event, window);
        let changed = !matches!(outcome, Ok(false));
        if let Err(cause) = outcome {
            self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
        }
        self.sketch_tool = Some(SketchToolState::Move(tool));
        changed
    }
    fn cancel_sketch_move(&mut self, edit: HeadSketchEdit) -> bool {
        self.command_notice = Some(match self.clear_geometric_picks(edit) {
            Ok(()) => DesktopCommandNotice::Ready,
            Err(cause) => DesktopCommandNotice::AuthoringFailed { cause },
        });
        true
    }
    fn move_event(
        &mut self,
        tool: &mut SketchMoveState,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        if window.navigation_active() {
            return Ok(false);
        }
        let Some((position, click)) = move_pointer(event) else {
            return Ok(false);
        };
        match tool.stage {
            MoveStage::Geometry if click => self.move_select(tool, position, window),
            MoveStage::Base(entity) if click => self.move_base(tool, entity, position, window),
            MoveStage::Target(entity, base) => {
                self.move_target(tool, entity, base, position, click, window)
            }
            _ => Ok(false),
        }
    }
    fn move_select(
        &mut self,
        tool: &mut SketchMoveState,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some(hit) = self.resolve_sketch_hit(
            tool.edit,
            pixel,
            window,
            super::sketch_picking::SketchPickFilter::Any,
        )?
        else {
            return Ok(false);
        };
        self.store_sketch_click(tool.edit, Some(hit.entity), SelectionClickMode::Replace)?;
        tool.stage = MoveStage::Base(hit.entity);
        self.placement_notice("Move: Select base point".into());
        Ok(true)
    }
    fn move_base(
        &mut self,
        tool: &mut SketchMoveState,
        entity: SketchEntityId,
        pixel: [i32; 2],
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some(base) = window.sketch_point(pixel, tool.edit.plane)? else {
            return Ok(false);
        };
        tool.stage = MoveStage::Target(entity, base);
        self.placement_notice("Move: Select end point".into());
        Ok(true)
    }
    fn move_target(
        &mut self,
        tool: &mut SketchMoveState,
        entity: SketchEntityId,
        base: [f64; 2],
        pixel: [i32; 2],
        click: bool,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        tool.preview = None;
        let Some(point) = window.sketch_point(pixel, tool.edit.plane)? else {
            return Ok(true);
        };
        let vector = std::array::from_fn(|axis| point[axis] - base[axis]);
        if vector == [0., 0.] {
            if click {
                return Err("move vector [0, 0]: expected distinct base and end points".into());
            }
            return Ok(true);
        }
        tool.preview = Some(self.preview_sketch_move(tool, entity, vector)?);
        if click {
            self.commit_sketch_move(tool, entity, vector)?;
        }
        Ok(true)
    }
    fn preview_sketch_move(
        &self,
        tool: &SketchMoveState,
        entity: SketchEntityId,
        vector: [f64; 2],
    ) -> Result<ViewportSketchPreview, String> {
        self.require_move_transaction_clear(tool.edit.document)?;
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|e| e.to_string())?;
        let preview = session
            .preview_sketch_transform(
                tool.edit.sketch,
                &[entity],
                SketchSelectionTransform::Move(vector),
            )
            .map_err(|e| e.to_string())?;
        Ok(ViewportSketchPreview::Resolved {
            plane: tool.edit.plane,
            geometry: Box::new(preview.geometry),
            members: preview.members,
        })
    }
    fn sketch_move_request(
        &self,
        edit: HeadSketchEdit,
        entity: SketchEntityId,
        vector: [f64; 2],
    ) -> Result<TransformSketchArgs, String> {
        let session = self
            .files
            .workspace()
            .session(edit.document)
            .map_err(|e| e.to_string())?;
        let snapshot = session.snapshot();
        let sketch_index =
            crate::sketch_geometric::wire_sketch_index(&snapshot.sketches, edit.sketch)?;
        let scale = snapshot.parameters.units().working_scale();
        Ok(TransformSketchArgs {
            sketch_index,
            op: "move".into(),
            entities: Some(vec![entity.0]),
            vector: Some(vector.map(|v| v * scale).to_vec()),
            ..Default::default()
        })
    }
    fn commit_sketch_move(
        &mut self,
        tool: &mut SketchMoveState,
        entity: SketchEntityId,
        vector: [f64; 2],
    ) -> Result<(), String> {
        let request = self.sketch_move_request(tool.edit, entity, vector)?;
        Client::new(self.document_host())
            .sketch()
            .transform(&request)
            .map_err(|e| e.to_string())?;
        tool.generation = self.capture_sketch_tool_context()?.1;
        self.clear_geometric_picks(tool.edit)?;
        tool.stage = MoveStage::Geometry;
        tool.preview = None;
        self.placement_notice("Move: Select geometry".into());
        Ok(())
    }
}

fn is_cancel(event: ViewportEvent) -> bool {
    matches!(
        event,
        ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Cancel,
            pressed: true
        })
    )
}
fn move_pointer(event: ViewportEvent) -> Option<([i32; 2], bool)> {
    match event {
        ViewportEvent::Input(ViewportInput::Motion { position, .. }) => Some((position, false)),
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed: true,
            position,
            modifiers,
        }) if !modifiers.alt && !modifiers.super_key => Some((position, true)),
        _ => None,
    }
}
