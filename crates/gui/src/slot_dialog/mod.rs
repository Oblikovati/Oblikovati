// SPDX-License-Identifier: GPL-2.0-only
//! Full v1.1 Properties panel for Part Modify Slot (rounded/rectangular, seats, clearance, end mill, center marks).
mod catalog;
mod controls;
mod diagram;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::{SessionPresets, draw_properties_header};
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{
    GuiAuthoringContext, GuiCommandState, GuiHoleSeatKind, GuiHoleTermination, GuiShellAction,
};
use catalog::{clearance_dimensions, parse_browser_plane, validate_fields};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    hole::{HoleFastenerSpec, HoleKind, HoleTerminationSurface},
    slot::{SlotCenterMarks, SlotShape},
};
pub use types::{
    GuiSlotAction, GuiSlotEditSeed, GuiSlotPick, GuiSlotPreview, GuiSlotPreviewState,
    GuiSlotRequest,
};

#[derive(Clone, Debug, PartialEq)]
pub(super) struct SlotPresetFields {
    pub(super) shape: SlotShape,
    pub(super) kind: HoleKind,
    pub(super) seat: GuiHoleSeatKind,
    pub(super) length: String,
    pub(super) width: String,
    pub(super) rotation: String,
    pub(super) depth: String,
    pub(super) seat_width: String,
    pub(super) seat_depth: String,
    pub(super) seat_angle: String,
    pub(super) termination: GuiHoleTermination,
    pub(super) end_mill_pointed: bool,
    pub(super) end_mill_angle: String,
    pub(super) center_marks: SlotCenterMarks,
    pub(super) direction_flipped: bool,
    pub(super) extend_start: bool,
    pub(super) imate: bool,
    pub(super) fastener: Option<HoleFastenerSpec>,
}

pub(crate) struct SlotDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    editing: Option<FeatureId>,
    body: Option<Vec<u8>>,
    pub(super) face: Option<Vec<u8>>,
    position: Option<[f64; 3]>,
    pub(super) shape: SlotShape,
    pub(super) kind: HoleKind,
    pub(super) seat: GuiHoleSeatKind,
    pub(super) length: String,
    pub(super) width: String,
    pub(super) rotation: String,
    pub(super) depth: String,
    pub(super) seat_width: String,
    pub(super) seat_depth: String,
    pub(super) seat_angle: String,
    pub(super) termination: GuiHoleTermination,
    pub(super) termination_surface: Option<HoleTerminationSurface>,
    pub(super) end_mill_pointed: bool,
    pub(super) end_mill_angle: String,
    pub(super) center_marks: SlotCenterMarks,
    pub(super) direction_flipped: bool,
    pub(super) extend_start: bool,
    pub(super) imate: bool,
    pub(super) fastener: Option<HoleFastenerSpec>,
    preview_hidden: bool,
    pub(super) input_geometry_open: bool,
    presets: SessionPresets<SlotPresetFields>,
    attempted: Option<GuiSlotRequest>,
    pub(super) submitted: Option<GuiSlotRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick: Option<GuiSlotPick>,
}

impl Default for SlotDialog {
    fn default() -> Self {
        let mut presets = SessionPresets::default();
        presets.shown = true;
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            body: None,
            face: None,
            position: None,
            shape: SlotShape::Rounded,
            kind: HoleKind::Simple,
            seat: GuiHoleSeatKind::None,
            length: "4 mm".into(),
            width: "2 mm".into(),
            rotation: "0 deg".into(),
            depth: "8 mm".into(),
            seat_width: "4 mm".into(),
            seat_depth: "2 mm".into(),
            seat_angle: "90 deg".into(),
            termination: GuiHoleTermination::ThroughAll,
            termination_surface: None,
            end_mill_pointed: false,
            end_mill_angle: "90 deg".into(),
            center_marks: SlotCenterMarks::None,
            direction_flipped: false,
            extend_start: false,
            imate: false,
            fastener: None,
            preview_hidden: false,
            input_geometry_open: true,
            presets,
            attempted: None,
            submitted: None,
            reopen_after_commit: false,
            consumed_pick: None,
        }
    }
}

impl DockableWindow for SlotDialog {}

impl SlotDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Slot", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, edit: Option<FeatureId>) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.preview_hidden = false;
        self.consumed_pick = state.slot_pick.clone();
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.slot_edits.get(&id))
        {
            self.presets.clear_selection();
            self.body = Some(seed.body.clone());
            self.face = Some(seed.face.clone());
            self.position = seed.position;
            self.shape = seed.shape;
            self.kind = seed.kind;
            self.seat = seed.seat;
            self.length = seed.length.clone();
            self.width = seed.width.clone();
            self.rotation = seed.rotation.clone();
            self.depth = seed.depth.clone();
            self.seat_width = seed.seat_width.clone();
            self.seat_depth = seed.seat_depth.clone();
            self.seat_angle = seed.seat_angle.clone();
            self.termination = seed.termination;
            self.termination_surface = seed.termination_surface.clone();
            self.end_mill_pointed = seed.end_mill_pointed;
            self.end_mill_angle = seed.end_mill_angle.clone();
            self.center_marks = seed.center_marks;
            self.direction_flipped = seed.direction_flipped;
            self.extend_start = seed.extend_start;
            self.imate = seed.imate;
            self.fastener = seed.fastener.clone();
            self.input_geometry_open = false;
            return;
        }
        self.body = None;
        self.face = state.selected_model_face.clone();
        self.position = None;
        self.input_geometry_open = true;
        if let Some(saved) = self.presets.initial_fields() {
            self.apply_preset(saved);
        } else {
            self.reset_defaults();
        }
    }

    fn reset_defaults(&mut self) {
        self.shape = SlotShape::Rounded;
        self.kind = HoleKind::Simple;
        self.seat = GuiHoleSeatKind::None;
        self.length = "4 mm".into();
        self.width = "2 mm".into();
        self.rotation = "0 deg".into();
        self.depth = "8 mm".into();
        self.seat_width = "4 mm".into();
        self.seat_depth = "2 mm".into();
        self.seat_angle = "90 deg".into();
        self.termination = GuiHoleTermination::ThroughAll;
        self.termination_surface = None;
        self.end_mill_pointed = false;
        self.end_mill_angle = "90 deg".into();
        self.center_marks = SlotCenterMarks::None;
        self.direction_flipped = false;
        self.extend_start = false;
        self.imate = false;
        self.fastener = None;
    }

    pub(super) fn apply_clearance_dimensions(&mut self, spec: &HoleFastenerSpec) {
        let (width, sw, sd) = clearance_dimensions(&spec.size, spec.fit);
        self.width = width.into();
        self.seat_width = sw.into();
        self.seat_depth = sd.into();
    }

    fn current_preset(&self) -> SlotPresetFields {
        SlotPresetFields {
            shape: self.shape,
            kind: self.kind,
            seat: self.seat,
            length: self.length.clone(),
            width: self.width.clone(),
            rotation: self.rotation.clone(),
            depth: self.depth.clone(),
            seat_width: self.seat_width.clone(),
            seat_depth: self.seat_depth.clone(),
            seat_angle: self.seat_angle.clone(),
            termination: self.termination,
            end_mill_pointed: self.end_mill_pointed,
            end_mill_angle: self.end_mill_angle.clone(),
            center_marks: self.center_marks,
            direction_flipped: self.direction_flipped,
            extend_start: self.extend_start,
            imate: self.imate,
            fastener: self.fastener.clone(),
        }
    }

    fn apply_preset(&mut self, preset: SlotPresetFields) {
        self.shape = preset.shape;
        self.kind = preset.kind;
        self.seat = preset.seat;
        self.length = preset.length;
        self.width = preset.width;
        self.rotation = preset.rotation;
        self.depth = preset.depth;
        self.seat_width = preset.seat_width;
        self.seat_depth = preset.seat_depth;
        self.seat_angle = preset.seat_angle;
        self.termination = preset.termination;
        self.end_mill_pointed = preset.end_mill_pointed;
        self.end_mill_angle = preset.end_mill_angle;
        self.center_marks = preset.center_marks;
        self.direction_flipped = preset.direction_flipped;
        self.extend_start = preset.extend_start;
        self.imate = preset.imate;
        self.fastener = preset.fastener;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Slot(GuiSlotAction::Clear));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
    }

    pub(crate) fn sync_before_docking(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_some()
            && (self.document != state.document || state.authoring != GuiAuthoringContext::Part)
        {
            self.close(actions);
            return;
        }
        if let Some(submitted) = &self.submitted
            && state.document_generation != Some(submitted.generation)
        {
            self.presets.record_accepted(self.current_preset());
            if self.reopen_after_commit {
                self.reopen_after_commit = false;
                self.open(state, None);
            } else {
                self.close(actions);
            }
        }
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiSlotRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiSlotRequest {
            document,
            generation,
            feature: self.editing,
            body: self.body.clone(),
            face: self.face.clone(),
            position: self.position,
            shape: self.shape,
            kind: self.kind,
            seat: self.seat,
            length: self.length.clone(),
            width: self.width.clone(),
            rotation: self.rotation.clone(),
            depth: self.depth.clone(),
            seat_width: self.seat_width.clone(),
            seat_depth: self.seat_depth.clone(),
            seat_angle: self.seat_angle.clone(),
            termination: self.termination,
            termination_surface: self.termination_surface.clone(),
            end_mill_pointed: self.end_mill_pointed,
            end_mill_angle: self.end_mill_angle.clone(),
            center_marks: self.center_marks,
            direction_flipped: self.direction_flipped,
            extend_start: self.extend_start,
            imate: self.imate,
            fastener: self.fastener.clone(),
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState, browser_plane: Option<&str>) {
        if let Some(plane_ref) = browser_plane
            && self.termination == GuiHoleTermination::ToSurface
            && let Some(surface) = parse_browser_plane(plane_ref)
        {
            self.termination_surface = Some(surface);
        }
        if self.editing.is_none() {
            if let Some(pick) = &state.slot_pick
                && self.consumed_pick.as_ref() != Some(pick)
            {
                self.consumed_pick = Some(pick.clone());
                self.body = Some(pick.body.clone());
                self.face = Some(pick.face.clone());
                self.position = pick.position;
            } else if self.face.is_none()
                && let Some(face) = &state.selected_model_face
            {
                self.face = Some(face.clone());
            }
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        icons: &GuiRibbonIcons,
        browser_plane: Option<&str>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state, browser_plane);
        let height = ((area.bottom - area.top) - 24.0).clamp(340.0, 660.0);
        widget_abi::fit_viewport_properties(ui, "Slot", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        let mut enter_commit = false;
        Self::dockable_window(ui, "Slot", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Slot",
                "##slot-adv",
                &mut self.presets.shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let had_popup = self.presets.popup_open;
            if let Some(chosen) = self.presets.draw(
                ui,
                icons,
                "slot",
                "New Slot Default",
                Some(self.current_preset()),
                "Slot Preset",
            ) {
                self.apply_preset(chosen);
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(120.0);
            ui.child_window("##slot-body")
                .size([0.0, body_h])
                .build(|| {
                    enter_commit = self.draw_body(ui, state);
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer_controls(
                ui,
                state,
                enter_commit,
                had_popup || self.presets.popup_open,
                actions,
            );
        });
        if !open {
            self.close(actions);
        }
    }

    fn draw_body(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        let mut enter_commit = false;
        let local_error = validate_fields(self);
        self.draw_input_geometry(ui);
        if let Some(error) = local_error {
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {error}"));
        } else if let Some(preview) = &state.slot_preview
            && let GuiSlotPreview::Failed(msg) | GuiSlotPreview::UnchangedCut(msg) = &preview.result
        {
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
        self.draw_shape_type_and_seat(ui, local_error.is_some());
        if self.kind == HoleKind::Clearance {
            self.draw_clearance_controls(ui);
        }
        self.draw_behavior(ui);
        diagram::draw_slot_diagram(
            ui,
            &diagram::SlotDiagramSpec {
                shape: self.shape,
                seat: self.seat,
                termination: self.termination,
                end_mill_pointed: self.end_mill_pointed,
                length: &self.length,
                width: &self.width,
                depth: &self.depth,
                seat_width: &self.seat_width,
                seat_depth: &self.seat_depth,
                seat_angle: &self.seat_angle,
                end_mill_angle: &self.end_mill_angle,
                invalid: local_error.is_some(),
            },
        );
        enter_commit |= self.draw_dimensions(ui, local_error);
        enter_commit
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Slot(GuiSlotAction::Preview(Box::new(req.clone()))).dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => {
                    actions.push(GuiShellAction::Slot(GuiSlotAction::Preview(Box::new(req))))
                }
                None => actions.push(GuiShellAction::Slot(GuiSlotAction::Clear)),
            }
        }
    }
}
