// SPDX-License-Identifier: GPL-2.0-only
//! Controls for Part Surface Ruled Surface: Type, Edges list, Automatic Edge Chain, Vector, Flip, Distance, Angle, and OK/Apply/Cancel (#3918).
use super::types::{GuiRuledSurfaceAction, GuiRuledSurfacePickTarget};
use super::{RuledSurfaceDialog, format_deg_angle};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::ruled_surface::RuledSurfaceType;

impl RuledSurfaceDialog {
    pub(super) fn draw_type_controls(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        ui.text("Type");
        let avail = ui.content_region_avail()[0].max(160.0);
        let btn_w = ((avail - 8.0) / 3.0).max(52.0);
        let types = [
            (RuledSurfaceType::Normal, "Normal"),
            (RuledSurfaceType::Tangent, "Tangent"),
            (RuledSurfaceType::Sweep, "Sweep"),
        ];
        for (idx, &(kind, label)) in types.iter().enumerate() {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            if toggle_button(
                ui,
                &format!("{label}##ruled-type-{idx}"),
                self.ruled_type == kind,
                [btn_w, 24.0],
            ) && self.ruled_type != kind
            {
                self.ruled_type = kind;
                if kind != RuledSurfaceType::Sweep {
                    self.pick_target = GuiRuledSurfacePickTarget::Edges;
                    actions.push(GuiShellAction::RuledSurface(
                        GuiRuledSurfaceAction::SetPickTarget(self.pick_target),
                    ));
                }
            }
        }
    }

    pub(super) fn draw_edge_controls(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Edges");
        let count = self.edges.len();
        let has_edges = count > 0;
        let edge_label = match count {
            0 => "Select Edges##ruled-edges-pill".to_string(),
            1 => "1 Edge##ruled-edges-pill".to_string(),
            n => format!("{n} Edges##ruled-edges-pill"),
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_edges { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_edges { 4.0 } else { 0.0 }).max(80.0);
        let active = self.pick_target == GuiRuledSurfacePickTarget::Edges;
        if selector_pill(ui, &edge_label, has_edges, active, [pill_w, 22.0]) {
            self.pick_target = GuiRuledSurfacePickTarget::Edges;
            actions.push(GuiShellAction::RuledSurface(
                GuiRuledSurfaceAction::SetPickTarget(self.pick_target),
            ));
        }
        if has_edges {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##ruled-edges-clear", [clear_w, 22.0]) {
                self.edges.clear();
                self.target_body = None;
            }
        }
        if !self.edges.is_empty() {
            let list_h = ((self.edges.len() as f32) * 20.0 + 10.0).min(96.0);
            ui.child_window("##ruled-edge-list")
                .size([0.0, list_h])
                .border(true)
                .build(|| {
                    for idx in 0..self.edges.len() {
                        ui.text(format!("Edge{}", idx + 1));
                    }
                });
        }
        let prev_chain = self.automatic_edge_chain;
        ui.checkbox("Automatic Edge Chain", &mut self.automatic_edge_chain);
        if self.automatic_edge_chain != prev_chain {
            actions.push(GuiShellAction::RuledSurface(
                GuiRuledSurfaceAction::SetAutomaticEdgeChain(self.automatic_edge_chain),
            ));
        }
    }

    pub(super) fn draw_vector_controls(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Vector");
        let has_vector = self.vector.is_some();
        let label = match &self.vector {
            Some(vec) => format!("{}##ruled-vector-pill", vec.label()),
            None => "Select Vector##ruled-vector-pill".to_string(),
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_vector { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_vector { 4.0 } else { 0.0 }).max(80.0);
        let active = self.pick_target == GuiRuledSurfacePickTarget::Vector;
        if selector_pill(ui, &label, has_vector, active, [pill_w, 22.0]) {
            self.pick_target = GuiRuledSurfacePickTarget::Vector;
            actions.push(GuiShellAction::RuledSurface(
                GuiRuledSurfaceAction::SetPickTarget(self.pick_target),
            ));
        }
        if has_vector {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##ruled-vector-clear", [clear_w, 22.0]) {
                self.vector = None;
            }
        }
    }

    pub(super) fn draw_dimension_controls(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Direction");
        let avail = ui.content_region_avail()[0].max(90.0);
        if toggle_button(ui, "Flip##ruled-flip", self.flip, [avail, 22.0]) {
            self.flip = !self.flip;
        }

        crate::theme::property_label(ui, "Distance");
        let dist_valid = self.parsed_distance_cm().is_some();
        let dist_w = (ui.content_region_avail()[0] - if dist_valid { 0.0 } else { 20.0 }).max(80.0);
        let dist_text_color = (!dist_valid).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Text,
                crate::theme::error_accent(ui),
            )
        });
        let dist_border = (!dist_valid).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Border,
                crate::theme::error_accent(ui),
            )
        });
        let dist_border_w =
            (!dist_valid).then(|| ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.5)));
        ui.set_next_item_width(dist_w);
        ui.input_text("##ruled-distance", &mut self.distance_text)
            .build();
        drop(dist_border_w);
        drop(dist_border);
        drop(dist_text_color);
        if !dist_valid {
            crate::feature_properties::draw_error_badge(ui, "Distance must be greater than 0");
        }

        crate::theme::property_label(ui, "Angle");
        let angle_valid = self.parsed_angle_deg().is_some();
        let angle_w =
            (ui.content_region_avail()[0] - if angle_valid { 0.0 } else { 20.0 }).max(80.0);
        let angle_text_color = (!angle_valid).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Text,
                crate::theme::error_accent(ui),
            )
        });
        let angle_border = (!angle_valid).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Border,
                crate::theme::error_accent(ui),
            )
        });
        let angle_border_w =
            (!angle_valid).then(|| ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.5)));
        ui.set_next_item_width(angle_w);
        ui.input_text("##ruled-angle", &mut self.angle_text).build();
        let angle_left = ui.is_item_deactivated_after_edit();
        drop(angle_border_w);
        drop(angle_border);
        drop(angle_text_color);
        if angle_left && let Some(deg) = self.parsed_angle_deg() {
            self.angle_text = format_deg_angle(deg);
        }
        if !angle_valid {
            crate::feature_properties::draw_error_badge(ui, "Angle must be between -180 and 180");
        }
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let local_valid = !self.edges.is_empty()
            && self.parsed_distance_cm().is_some()
            && self.parsed_angle_deg().is_some()
            && (self.ruled_type != RuledSurfaceType::Sweep || self.vector.is_some());
        let ready_request = local_valid
            .then(|| self.request(state))
            .flatten()
            .filter(|req| {
                GuiShellAction::RuledSurface(GuiRuledSurfaceAction::Commit(Box::new(req.clone())))
                    .dispatchable(state)
            });

        if crate::theme::property_accept(ui, ready_request.is_some())
            && let Some(req) = ready_request.clone()
        {
            self.submitted = Some(req.clone());
            self.apply_keep_open = false;
            actions.push(GuiShellAction::RuledSurface(GuiRuledSurfaceAction::Commit(
                Box::new(req),
            )));
        }
        ui.same_line_with_spacing(0.0, 6.0);
        let apply_disabled = ui.begin_disabled(ready_request.is_none());
        if widget_abi::button(ui, "Apply", [64.0, 0.0])
            && let Some(req) = ready_request
        {
            self.submitted = Some(req.clone());
            self.apply_keep_open = true;
            actions.push(GuiShellAction::RuledSurface(GuiRuledSurfaceAction::Commit(
                Box::new(req),
            )));
        }
        drop(apply_disabled);
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
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
