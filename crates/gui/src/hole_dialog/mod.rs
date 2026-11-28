// SPDX-License-Identifier: GPL-2.0-only
//! Full v1.1 Properties panel for Part Modify Hole (simple, clearance, tapped, taper tapped, seats, drill points, presets).
mod catalog;
mod controls;
mod diagram;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::{SessionPresets, draw_properties_header};
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use catalog::{clearance_dimensions, parse_browser_plane, validate_fields};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    hole::{HoleFastenerSpec, HoleKind, HoleTerminationSurface, HoleThreadSpec},
};
pub use types::{
    GuiHoleAction, GuiHoleEditSeed, GuiHolePick, GuiHolePreview, GuiHolePreviewState,
    GuiHoleRequest, GuiHoleSeatKind, GuiHoleTermination,
};

#[derive(Clone, Debug, PartialEq)]
pub(super) struct HolePresetFields {
    pub(super) kind: HoleKind,
    pub(super) seat: GuiHoleSeatKind,
    pub(super) diameter: String,
    pub(super) depth: String,
    pub(super) seat_diameter: String,
    pub(super) seat_depth: String,
    pub(super) seat_angle: String,
    pub(super) termination: GuiHoleTermination,
    pub(super) drill_point_angled: bool,
    pub(super) drill_point_angle: String,
    pub(super) direction_flipped: bool,
    pub(super) extend_start: bool,
    pub(super) fastener: Option<HoleFastenerSpec>,
    pub(super) thread: Option<HoleThreadSpec>,
}

pub(crate) struct HoleDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    editing: Option<FeatureId>,
    body: Option<Vec<u8>>,
    pub(super) face: Option<Vec<u8>>,
    position: Option<[f64; 3]>,
    pub(super) kind: HoleKind,
    pub(super) seat: GuiHoleSeatKind,
    pub(super) diameter: String,
    pub(super) depth: String,
    pub(super) seat_diameter: String,
    pub(super) seat_depth: String,
    pub(super) seat_angle: String,
    pub(super) termination: GuiHoleTermination,
    pub(super) termination_surface: Option<HoleTerminationSurface>,
    pub(super) drill_point_angled: bool,
    pub(super) drill_point_angle: String,
    pub(super) direction_flipped: bool,
    pub(super) extend_start: bool,
    pub(super) fastener: Option<HoleFastenerSpec>,
    pub(super) thread: Option<HoleThreadSpec>,
    preview_hidden: bool,
    pub(super) input_geometry_open: bool,
    presets: SessionPresets<HolePresetFields>,
    attempted: Option<GuiHoleRequest>,
    pub(super) submitted: Option<GuiHoleRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick: Option<GuiHolePick>,
}

impl Default for HoleDialog {
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
            kind: HoleKind::Simple,
            seat: GuiHoleSeatKind::None,
            diameter: "4 mm".into(),
            depth: "8 mm".into(),
            seat_diameter: "6 mm".into(),
            seat_depth: "2 mm".into(),
            seat_angle: "90 deg".into(),
            termination: GuiHoleTermination::ThroughAll,
            termination_surface: None,
            drill_point_angled: false,
            drill_point_angle: "118 deg".into(),
            direction_flipped: false,
            extend_start: true,
            fastener: None,
            thread: None,
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

impl DockableWindow for HoleDialog {}

impl HoleDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Hole", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, edit: Option<FeatureId>) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.preview_hidden = false;
        self.consumed_pick = state.hole_pick.clone();
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.hole_edits.get(&id))
        {
            self.presets.clear_selection();
            self.body = Some(seed.body.clone());
            self.face = Some(seed.face.clone());
            self.position = seed.position;
            self.kind = seed.kind;
            self.seat = seed.seat;
            self.diameter = seed.diameter.clone();
            self.depth = seed.depth.clone();
            self.seat_diameter = seed.seat_diameter.clone();
            self.seat_depth = seed.seat_depth.clone();
            self.seat_angle = seed.seat_angle.clone();
            self.termination = seed.termination;
            self.termination_surface = seed.termination_surface.clone();
            self.drill_point_angled = seed.drill_point_angled;
            self.drill_point_angle = seed.drill_point_angle.clone();
            self.direction_flipped = seed.direction_flipped;
            self.extend_start = seed.extend_start;
            self.fastener = seed.fastener.clone();
            self.thread = seed.thread.clone();
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
        self.kind = HoleKind::Simple;
        self.seat = GuiHoleSeatKind::None;
        self.diameter = "4 mm".into();
        self.depth = "8 mm".into();
        self.seat_diameter = "6 mm".into();
        self.seat_depth = "2 mm".into();
        self.seat_angle = "90 deg".into();
        self.termination = GuiHoleTermination::ThroughAll;
        self.termination_surface = None;
        self.drill_point_angled = false;
        self.drill_point_angle = "118 deg".into();
        self.direction_flipped = false;
        self.extend_start = true;
        self.fastener = None;
        self.thread = None;
    }

    pub(super) fn apply_clearance_dimensions(&mut self, spec: &HoleFastenerSpec) {
        let (bore, sd, sdepth) = clearance_dimensions(&spec.size, spec.fit);
        self.diameter = bore.into();
        self.seat_diameter = sd.into();
        self.seat_depth = sdepth.into();
    }

    fn current_preset(&self) -> HolePresetFields {
        HolePresetFields {
            kind: self.kind,
            seat: self.seat,
            diameter: self.diameter.clone(),
            depth: self.depth.clone(),
            seat_diameter: self.seat_diameter.clone(),
            seat_depth: self.seat_depth.clone(),
            seat_angle: self.seat_angle.clone(),
            termination: self.termination,
            drill_point_angled: self.drill_point_angled,
            drill_point_angle: self.drill_point_angle.clone(),
            direction_flipped: self.direction_flipped,
            extend_start: self.extend_start,
            fastener: self.fastener.clone(),
            thread: self.thread.clone(),
        }
    }

    fn apply_preset(&mut self, preset: HolePresetFields) {
        self.kind = preset.kind;
        self.seat = preset.seat;
        self.diameter = preset.diameter;
        self.depth = preset.depth;
        self.seat_diameter = preset.seat_diameter;
        self.seat_depth = preset.seat_depth;
        self.seat_angle = preset.seat_angle;
        self.termination = preset.termination;
        self.drill_point_angled = preset.drill_point_angled;
        self.drill_point_angle = preset.drill_point_angle;
        self.direction_flipped = preset.direction_flipped;
        self.extend_start = preset.extend_start;
        self.fastener = preset.fastener;
        self.thread = preset.thread;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Hole(GuiHoleAction::Clear));
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
            && (self.document != state.document
                || !state.fillet_available
                || state.authoring != GuiAuthoringContext::Part)
        {
            self.close(actions);
            return;
        }
        if let Some(submitted) = &self.submitted
            && state
                .fillet_selection
                .as_ref()
                .is_none_or(|(generation, _)| *generation != submitted.generation)
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

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiHoleRequest> {
        let document = self.document?;
        let (generation, _) = state.fillet_selection.as_ref()?;
        Some(GuiHoleRequest {
            document,
            generation: *generation,
            feature: self.editing,
            body: self.body.clone(),
            face: self.face.clone(),
            position: self.position,
            kind: self.kind,
            seat: self.seat,
            diameter: self.diameter.clone(),
            depth: self.depth.clone(),
            seat_diameter: self.seat_diameter.clone(),
            seat_depth: self.seat_depth.clone(),
            seat_angle: self.seat_angle.clone(),
            termination: self.termination,
            termination_surface: self.termination_surface.clone(),
            drill_point_angled: self.drill_point_angled,
            drill_point_angle: self.drill_point_angle.clone(),
            direction_flipped: self.direction_flipped,
            extend_start: self.extend_start,
            fastener: self.fastener.clone(),
            thread: self.thread.clone(),
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
            if let Some(pick) = &state.hole_pick
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
        let height = ((area.bottom - area.top) - 28.0).clamp(320.0, 620.0);
        widget_abi::fit_viewport_properties(ui, "Hole", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        let mut enter_commit = false;
        Self::dockable_window(ui, "Hole", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Hole",
                "##hole-adv",
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
                "hole",
                "New Hole Default",
                Some(self.current_preset()),
                "Hole Preset",
            ) {
                self.apply_preset(chosen);
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(120.0);
            ui.child_window("##hole-body")
                .size([0.0, body_h])
                .build(|| {
                    enter_commit = self.draw_body(ui, state);
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer(
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
        self.draw_type_and_seat(ui, local_error.is_some());
        if let Some(error) = local_error {
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {error}"));
        } else if let Some(preview) = &state.hole_preview
            && let GuiHolePreview::Failed(msg) | GuiHolePreview::UnchangedCut(msg) = &preview.result
        {
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
        if self.kind == HoleKind::Clearance {
            self.draw_clearance_controls(ui);
        } else if matches!(self.kind, HoleKind::Tapped | HoleKind::TaperTapped) {
            self.draw_thread_controls(ui);
        }
        self.draw_behavior(ui);
        diagram::draw_hole_diagram(
            ui,
            &diagram::HoleDiagramSpec {
                seat: self.seat,
                termination: self.termination,
                drill_point_angled: self.drill_point_angled,
                diameter: &self.diameter,
                depth: &self.depth,
                seat_diameter: &self.seat_diameter,
                seat_depth: &self.seat_depth,
                seat_angle: &self.seat_angle,
                drill_point_angle: &self.drill_point_angle,
                invalid: local_error.is_some(),
            },
        );
        enter_commit |= self.draw_dimensions(ui, local_error);
        enter_commit
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Hole(GuiHoleAction::Preview(Box::new(req.clone()))).dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => {
                    actions.push(GuiShellAction::Hole(GuiHoleAction::Preview(Box::new(req))))
                }
                None => actions.push(GuiShellAction::Hole(GuiHoleAction::Clear)),
            }
        }
    }

    fn draw_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        enter_commit: bool,
        popup_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.draw_footer_controls(ui, state, enter_commit, popup_active, actions);
    }
}
