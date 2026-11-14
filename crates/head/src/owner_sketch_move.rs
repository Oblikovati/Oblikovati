// SPDX-License-Identifier: GPL-2.0-only
//! Canvas selection, solved preview, and typed commit for the sketch Move command.
use super::*;
use crate::{HeadSketchEdit, HeadWindow, sketch_tool::SketchToolState};
use oblikovati_api::{client::Client, wire::TransformSketchArgs};
use oblikovati_gui::{
    GuiPrecisePoint, GuiSketchMoveAction, GuiSketchMoveStage, GuiSketchMoveStatus,
};
use oblikovati_model::sketch::{SketchEntityId, SketchSelectionTransform};
use oblikovati_session::selection::SelectionClickMode;
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
    ViewportSketchPreview,
};

#[derive(Clone, Copy)]
enum MoveStage {
    Geometry,
    Base,
    Target([f64; 2]),
}
pub(crate) struct SketchMoveState {
    pub edit: HeadSketchEdit,
    pub generation: u64,
    stage: MoveStage,
    entities: Vec<SketchEntityId>,
    pointer: Option<[f64; 2]>,
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
            GuiSketchMoveAction::SetOptimizeSingle(value) => self.set_move_optimize_single(value),
            GuiSketchMoveAction::SetCopy(value) => self.set_move_copy(value),
            GuiSketchMoveAction::ChooseBase => self.choose_move_base(),
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
        self.sketch_memory.triad_pick = None;
        self.sketch_tool = Some(SketchToolState::Move(Box::new(SketchMoveState {
            edit,
            generation,
            stage: MoveStage::Geometry,
            entities: Vec::new(),
            pointer: None,
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
    pub(super) fn gui_sketch_move_stage(&self) -> Option<GuiSketchMoveStatus> {
        let Some(SketchToolState::Move(tool)) = &self.sketch_tool else {
            return None;
        };
        if !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation) {
            return None;
        }
        let stage = match tool.stage {
            MoveStage::Geometry => GuiSketchMoveStage::Geometry,
            MoveStage::Base => GuiSketchMoveStage::Base,
            MoveStage::Target(_) => GuiSketchMoveStage::Target,
        };
        Some(GuiSketchMoveStatus {
            stage,
            selection_count: tool.entities.len(),
            optimize_single: self.sketch_memory.move_optimize_single,
            copy: self.sketch_memory.move_copy,
        })
    }
    fn set_move_copy(&mut self, value: bool) -> Result<bool, String> {
        let Some(SketchToolState::Move(tool)) = &self.sketch_tool else {
            return Err(format!("move copy {value}: expected an active Move tool"));
        };
        if !matches!(tool.stage, MoveStage::Geometry) {
            return Err(format!(
                "move copy {value}: expected geometry selection stage"
            ));
        }
        self.sketch_memory.move_copy = value;
        Ok(true)
    }
    fn set_move_optimize_single(&mut self, value: bool) -> Result<bool, String> {
        let Some(SketchToolState::Move(tool)) = &self.sketch_tool else {
            return Err(format!(
                "move optimization {value}: expected an active Move tool"
            ));
        };
        if !matches!(tool.stage, MoveStage::Geometry) {
            return Err(format!(
                "move optimization {value}: expected geometry selection stage"
            ));
        }
        self.sketch_memory.move_optimize_single = value;
        Ok(true)
    }
    fn choose_move_base(&mut self) -> Result<bool, String> {
        let action = oblikovati_gui::GuiShellAction::SketchMove(GuiSketchMoveAction::ChooseBase);
        if !action.enabled(&self.gui_command_state()?) {
            return Err(
                "move base: expected selected geometry in an active sketch without a transaction"
                    .into(),
            );
        }
        let Some(SketchToolState::Move(tool)) = self.sketch_tool.as_mut() else {
            return Err("move base: expected an active Move tool".into());
        };
        tool.stage = MoveStage::Base;
        self.placement_notice("Move: Select base point".into());
        Ok(true)
    }
    pub(super) fn process_sketch_move(
        &mut self,
        mut tool: Box<SketchMoveState>,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> bool {
        if self.sketch_memory.triad_pick.is_some() {
            let result = self.move_triad_event(&tool, event, window);
            if let Err(cause) = result {
                self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
            }
            self.sketch_tool = Some(SketchToolState::Move(tool));
            return true;
        }
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
            MoveStage::Base => self.move_base(tool, position, click, window),
            MoveStage::Target(base) => self.move_target(tool, base, position, click, window),
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
        let single = self.sketch_memory.move_optimize_single;
        let mode = if single {
            SelectionClickMode::Replace
        } else {
            SelectionClickMode::Toggle
        };
        self.store_sketch_click(tool.edit, Some(hit.entity), mode)?;
        if single {
            tool.entities.clear();
            tool.entities.push(hit.entity);
            tool.stage = MoveStage::Base;
            self.placement_notice("Move: Select base point".into());
        } else if let Some(index) = tool.entities.iter().position(|id| *id == hit.entity) {
            tool.entities.remove(index);
        } else {
            tool.entities.push(hit.entity);
        }
        Ok(true)
    }
    fn move_base(
        &mut self,
        tool: &mut SketchMoveState,
        pixel: [i32; 2],
        click: bool,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        let Some(base) = window.sketch_point(pixel, tool.edit.plane)? else {
            return Ok(false);
        };
        tool.pointer = Some(base);
        if !click {
            return Ok(true);
        }
        self.set_move_base(tool, base);
        Ok(true)
    }
    fn set_move_base(&mut self, tool: &mut SketchMoveState, base: [f64; 2]) {
        tool.stage = MoveStage::Target(base);
        self.sketch_memory.triad = Some(base);
        self.placement_notice("Move: Select end point".into());
    }
    fn move_target(
        &mut self,
        tool: &mut SketchMoveState,
        base: [f64; 2],
        pixel: [i32; 2],
        click: bool,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        tool.preview = None;
        let Some(point) = window.sketch_point(pixel, tool.edit.plane)? else {
            return Ok(true);
        };
        tool.pointer = Some(point);
        self.move_to_point(tool, base, point, click)
    }
    fn move_to_point(
        &mut self,
        tool: &mut SketchMoveState,
        base: [f64; 2],
        point: [f64; 2],
        click: bool,
    ) -> Result<bool, String> {
        tool.preview = None;
        let vector = std::array::from_fn(|axis| point[axis] - base[axis]);
        if vector == [0., 0.] {
            if click {
                return Err("move vector [0, 0]: expected distinct base and end points".into());
            }
            return Ok(true);
        }
        tool.preview = Some(self.preview_sketch_move(tool, vector)?);
        if click {
            self.commit_sketch_move(tool, vector)?;
        }
        Ok(true)
    }
    pub(super) fn move_precise_target(&self) -> Option<(HeadSketchEdit, Option<[f64; 2]>)> {
        let Some(SketchToolState::Move(tool)) = &self.sketch_tool else {
            return None;
        };
        if matches!(tool.stage, MoveStage::Geometry)
            || !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation)
        {
            return None;
        }
        Some((tool.edit, tool.pointer))
    }
    pub(super) fn precise_move_point(
        &mut self,
        tool: &mut SketchMoveState,
        point: &GuiPrecisePoint,
    ) -> Result<(), String> {
        if !self.sketch_context_valid(tool.edit.document, tool.edit.sketch, tool.generation) {
            return Err(format!(
                "precise move {point:?}: expected an unchanged sketch"
            ));
        }
        let position = self.precise_position(tool.edit.document, point)?;
        match tool.stage {
            MoveStage::Base => self.set_move_base(tool, position),
            MoveStage::Target(base) => {
                self.move_to_point(tool, base, position, true)?;
            }
            MoveStage::Geometry => {
                return Err(format!(
                    "precise move {point:?}: expected a base or end point"
                ));
            }
        }
        Ok(())
    }
    fn move_triad_event(
        &mut self,
        tool: &SketchMoveState,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<(), String> {
        use super::owner_precise_input::TriadPick;
        if is_cancel(event) {
            self.sketch_memory.triad_pick = None;
        } else if let Some((pixel, true)) = move_pointer(event) {
            self.sketch_memory.triad_pick = match self.sketch_memory.triad_pick {
                Some(TriadPick::Source) => Some(TriadPick::Target),
                Some(TriadPick::Target) => {
                    self.sketch_memory.triad = window.sketch_point(pixel, tool.edit.plane)?;
                    None
                }
                None => None,
            };
        }
        let prompt = match (self.sketch_memory.triad_pick, tool.stage) {
            (Some(TriadPick::Source), _) => "Select the triad's origin",
            (Some(TriadPick::Target), _) => "Select the triad's new position",
            (None, MoveStage::Geometry) => "Move: Select geometry",
            (None, MoveStage::Base) => "Move: Select base point",
            (None, MoveStage::Target(_)) => "Move: Select end point",
        };
        self.placement_notice(prompt.into());
        Ok(())
    }
    fn preview_sketch_move(
        &self,
        tool: &SketchMoveState,
        vector: [f64; 2],
    ) -> Result<ViewportSketchPreview, String> {
        self.require_move_transaction_clear(tool.edit.document)?;
        let session = self
            .files
            .workspace()
            .session(tool.edit.document)
            .map_err(|e| e.to_string())?;
        let preview = if self.sketch_memory.move_copy {
            session
                .preview_sketch_copy(tool.edit.sketch, &tool.entities, vector)
                .map_err(|e| e.to_string())?
        } else {
            session
                .preview_sketch_transform(
                    tool.edit.sketch,
                    &tool.entities,
                    SketchSelectionTransform::Move(vector),
                )
                .map_err(|e| e.to_string())?
        };
        Ok(ViewportSketchPreview::Resolved {
            plane: tool.edit.plane,
            geometry: Box::new(preview.geometry),
            members: preview.members,
        })
    }
    fn sketch_move_request(
        &self,
        edit: HeadSketchEdit,
        entities: &[SketchEntityId],
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
            op: if self.sketch_memory.move_copy {
                "copy"
            } else {
                "move"
            }
            .into(),
            entities: Some(entities.iter().map(|id| id.0).collect()),
            vector: Some(vector.map(|v| v * scale).to_vec()),
            ..Default::default()
        })
    }
    fn commit_sketch_move(
        &mut self,
        tool: &mut SketchMoveState,
        vector: [f64; 2],
    ) -> Result<(), String> {
        let request = self.sketch_move_request(tool.edit, &tool.entities, vector)?;
        Client::new(self.document_host())
            .sketch()
            .transform(&request)
            .map_err(|e| e.to_string())?;
        tool.generation = self.capture_sketch_tool_context()?.1;
        self.clear_geometric_picks(tool.edit)?;
        tool.stage = MoveStage::Geometry;
        tool.entities.clear();
        tool.preview = None;
        tool.pointer = None;
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
        }) | ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::Escape,
            pressed: true,
            ..
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
