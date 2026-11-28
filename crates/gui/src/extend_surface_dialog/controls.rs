// SPDX-License-Identifier: GPL-2.0-only
//! Controls for Part Surface Extend: Edges, Edge Chain, Extents (Distance/To), Method (Extend/Stretch), and OK/Cancel (#3918).
use super::ExtendSurfaceDialog;
use super::types::{GuiExtendSurfaceAction, GuiExtendSurfaceExtentMode, GuiExtendSurfacePreview};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::extend_surface::{
    ExtendSurfaceMethod, ExtendSurfaceOriginPlane, ExtendSurfaceTermination,
};

impl ExtendSurfaceDialog {
    pub(super) fn draw_input_geometry(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let header_label = if self.input_geometry_open {
            "v Input Geometry"
        } else {
            "> Input Geometry"
        };
        if widget_abi::selectable(ui, header_label, false, false) {
            self.input_geometry_open = !self.input_geometry_open;
        }
        if !self.input_geometry_open {
            return;
        }
        crate::theme::property_label(ui, "Edges");
        let count = self.edges.len();
        let has_edges = count > 0;
        let edge_label = match count {
            0 => "Select Edges##extend-edges-pill".to_string(),
            1 => "1 Edge##extend-edges-pill".to_string(),
            n => format!("{n} Edges##extend-edges-pill"),
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_edges { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_edges { 4.0 } else { 0.0 }).max(80.0);
        if selector_pill(
            ui,
            &edge_label,
            has_edges,
            !self.picking_termination,
            [pill_w, 22.0],
        ) {
            self.set_picking_termination(false, actions);
        }
        if has_edges {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##extend-edges-clear", [clear_w, 22.0]) {
                self.edges.clear();
                self.target_body = None;
                self.set_picking_termination(false, actions);
            }
        }
        let prev_chain = self.edge_chain;
        ui.checkbox("Edge Chain", &mut self.edge_chain);
        if self.edge_chain != prev_chain {
            actions.push(GuiShellAction::ExtendSurface(
                GuiExtendSurfaceAction::SetEdgeChain(self.edge_chain),
            ));
        }
    }

    pub(super) fn draw_extents_controls(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        ui.text("Extents");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(60.0);
        if toggle_button(
            ui,
            "Distance##extend-extent-distance",
            self.extent_mode == GuiExtendSurfaceExtentMode::Distance,
            [half, 22.0],
        ) {
            self.extent_mode = GuiExtendSurfaceExtentMode::Distance;
            self.set_picking_termination(false, actions);
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "To##extend-extent-to",
            self.extent_mode == GuiExtendSurfaceExtentMode::To,
            [half, 22.0],
        ) {
            self.extent_mode = GuiExtendSurfaceExtentMode::To;
            self.set_picking_termination(true, actions);
        }
        match self.extent_mode {
            GuiExtendSurfaceExtentMode::Distance => self.draw_distance_field(ui),
            GuiExtendSurfaceExtentMode::To => self.draw_termination_selector(ui, actions),
        }
    }

    fn draw_distance_field(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Distance");
        let dist_valid = self.parsed_distance_cm().is_some();
        let field_w =
            (ui.content_region_avail()[0] - if dist_valid { 0.0 } else { 20.0 }).max(80.0);
        let border_color = (!dist_valid).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Border,
                crate::theme::error_accent(ui),
            )
        });
        let border_w =
            (!dist_valid).then(|| ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.5)));
        ui.set_next_item_width(field_w);
        ui.input_text("##extend-distance", &mut self.distance_text)
            .build();
        drop(border_w);
        drop(border_color);
        if !dist_valid {
            crate::feature_properties::draw_error_badge(ui, "Invalid distance");
        }
    }

    fn draw_termination_selector(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Termination");
        let has_target = self.termination.is_some();
        let caption = match &self.termination {
            None => "Select Termination##extend-term-pill".to_string(),
            Some(ExtendSurfaceTermination::Face { .. }) => "1 Face##extend-term-pill".to_string(),
            Some(ExtendSurfaceTermination::WorkPlane(_)) => {
                "Work Plane##extend-term-pill".to_string()
            }
            Some(ExtendSurfaceTermination::OriginPlane(origin)) => {
                format!("{}##extend-term-pill", origin_label(*origin))
            }
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_target { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_target { 4.0 } else { 0.0 }).max(80.0);
        if selector_pill(
            ui,
            &caption,
            has_target,
            self.picking_termination,
            [pill_w, 22.0],
        ) {
            self.set_picking_termination(true, actions);
        }
        if has_target {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##extend-term-clear", [clear_w, 22.0]) {
                self.termination = None;
                self.set_picking_termination(true, actions);
            }
        }
    }

    pub(super) fn draw_more_controls(&mut self, ui: &Ui) {
        let header_label = if self.more_open { "v More" } else { "> More" };
        if widget_abi::selectable(ui, header_label, false, false) {
            self.more_open = !self.more_open;
        }
        if !self.more_open {
            return;
        }
        crate::theme::property_label(ui, "Method");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(60.0);
        if toggle_button(
            ui,
            "Extend##extend-method-extend",
            self.method == ExtendSurfaceMethod::Extend,
            [half, 22.0],
        ) {
            self.method = ExtendSurfaceMethod::Extend;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Stretch##extend-method-stretch",
            self.method == ExtendSurfaceMethod::Stretch,
            [half, 22.0],
        ) {
            self.method = ExtendSurfaceMethod::Stretch;
        }
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let ready_request = self.request(state).filter(|req| {
            GuiShellAction::ExtendSurface(GuiExtendSurfaceAction::Commit(Box::new(req.clone())))
                .dispatchable(state)
        });
        if crate::theme::property_accept(ui, ready_request.is_some())
            && let Some(req) = ready_request
        {
            self.submitted = Some(req.clone());
            actions.push(GuiShellAction::ExtendSurface(
                GuiExtendSurfaceAction::Commit(Box::new(req)),
            ));
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }

    pub(super) fn draw_status_error(&self, ui: &Ui, state: &GuiCommandState) {
        if let Some(preview) = &state.extend_surface_preview
            && let GuiExtendSurfacePreview::Failed(msg) = &preview.result
        {
            ui.spacing();
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
    }
}

fn origin_label(origin: ExtendSurfaceOriginPlane) -> &'static str {
    match origin {
        ExtendSurfaceOriginPlane::Xy => "XY Plane",
        ExtendSurfaceOriginPlane::Xz => "XZ Plane",
        ExtendSurfaceOriginPlane::Yz => "YZ Plane",
    }
}

fn selector_pill(ui: &Ui, label: &str, selected: bool, active: bool, size: [f32; 2]) -> bool {
    let color = if active {
        [0.24, 0.44, 0.78, 1.0]
    } else if selected {
        [0.20, 0.34, 0.56, 1.0]
    } else {
        [0.28, 0.28, 0.31, 1.0]
    };
    let _bg = widget_abi::push_style_color(ui, imgui::StyleColor::Button, color);
    widget_abi::button(ui, label, size)
}

fn toggle_button(ui: &Ui, label: &str, selected: bool, size: [f32; 2]) -> bool {
    let color = selected.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Button, [0.26, 0.36, 0.68, 1.0])
    });
    let clicked = widget_abi::button(ui, label, size);
    drop(color);
    clicked
}
