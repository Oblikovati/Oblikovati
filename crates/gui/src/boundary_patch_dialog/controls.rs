// SPDX-License-Identifier: GPL-2.0-only
//! Properties panel controls for Surface Boundary Patch (#3918).
use super::types::{
    GuiBoundaryCondition, GuiBoundaryPatchAction, GuiBoundaryPatchPreview,
    GuiBoundaryPatchSelectionMode,
};
use super::{BoundaryPatchDialog, format_weight};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

impl BoundaryPatchDialog {
    pub(super) fn draw_boundary_section(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let header = if self.boundary_open {
            "v Boundary"
        } else {
            "> Boundary"
        };
        if widget_abi::selectable(ui, header, false, false) {
            self.boundary_open = !self.boundary_open;
        }
        if !self.boundary_open {
            return;
        }
        let prev_chain = self.automatic_edge_chain;
        ui.checkbox("Automatic Edge Chain", &mut self.automatic_edge_chain);
        if self.automatic_edge_chain != prev_chain {
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::SetAutomaticEdgeChain(self.automatic_edge_chain),
            ));
        }
        let avail = ui.content_region_avail()[0].max(160.0);
        let boundary_active = self.selection_mode == GuiBoundaryPatchSelectionMode::Boundary;
        if self.edge_sets.is_empty() {
            if selector_pill(
                ui,
                "Edges1: Select Edges##bp-edges-empty",
                false,
                boundary_active,
                [avail, 22.0],
            ) {
                self.set_selection_mode(GuiBoundaryPatchSelectionMode::Boundary, actions);
            }
            return;
        }
        let mut clear_index = None;
        let count = self.edge_sets.len();
        for idx in 0..count {
            let edge_count = self.edge_sets[idx].edges.len();
            let count_text = if edge_count == 1 {
                "1 Edge".to_string()
            } else {
                format!("{edge_count} Edges")
            };
            let set_name = format!("Edges{}", idx + 1);
            let clear_w = 46.0;
            let pill_w = (avail - clear_w - 4.0).max(96.0);
            let label = format!("{set_name}: {count_text}##bp-set-{idx}");
            if selector_pill(ui, &label, true, boundary_active, [pill_w, 22.0]) {
                self.set_selection_mode(GuiBoundaryPatchSelectionMode::Boundary, actions);
            }
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, &format!("Clear##bp-set-clear-{idx}"), [clear_w, 22.0]) {
                clear_index = Some(idx);
            }
            self.draw_condition_row(ui, idx);
        }
        if let Some(idx) = clear_index {
            self.edge_sets.remove(idx);
        }
    }

    fn draw_condition_row(&mut self, ui: &Ui, idx: usize) {
        let Some(set) = self.edge_sets.get_mut(idx) else {
            return;
        };
        crate::theme::property_label(ui, "Condition");
        let cond_choices = [
            (GuiBoundaryCondition::Free, "G0", "Contact (G0)"),
            (GuiBoundaryCondition::Tangent, "G1", "Tangent (G1)"),
            (GuiBoundaryCondition::Smooth, "G2", "Smooth (G2)"),
        ];
        let btn_w = 34.0;
        for (slot, (cond, short, tip)) in cond_choices.into_iter().enumerate() {
            if slot > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            let selected = set.condition == cond;
            if toggle_button(
                ui,
                &format!("{short}##bp-cond-{idx}-{slot}"),
                selected,
                [btn_w, 22.0],
            ) {
                set.condition = cond;
                if cond.supports_weight() && set.weight_text.trim().is_empty() {
                    set.weight_text = format_weight(1.0);
                }
            }
            if ui.is_item_hovered() {
                ui.tooltip_text(tip);
            }
        }
        crate::theme::property_label(ui, "Weight");
        let weight_enabled = set.condition.supports_weight();
        let valid_weight = !weight_enabled || set.parsed_weight().is_some();
        let field_w =
            (ui.content_region_avail()[0] - if valid_weight { 0.0 } else { 20.0 }).max(80.0);
        let _disabled = ui.begin_disabled(!weight_enabled);
        let border_color = (!valid_weight).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Border,
                crate::theme::error_accent(ui),
            )
        });
        let border_w =
            (!valid_weight).then(|| ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.5)));
        ui.set_next_item_width(field_w);
        ui.input_text(&format!("##bp-weight-{idx}"), &mut set.weight_text)
            .build();
        drop(border_w);
        drop(border_color);
        drop(_disabled);
        if !valid_weight {
            crate::feature_properties::draw_error_badge(ui, "Invalid weight");
        }
    }

    pub(super) fn draw_guide_rails_section(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let header = if self.guide_rails_open {
            "v Guide Rails"
        } else {
            "> Guide Rails"
        };
        if widget_abi::selectable(ui, header, false, false) {
            self.guide_rails_open = !self.guide_rails_open;
        }
        if !self.guide_rails_open {
            return;
        }
        let avail = ui.content_region_avail()[0].max(160.0);
        let active = self.selection_mode == GuiBoundaryPatchSelectionMode::GuideRails;
        let has_rails = !self.guide_rails.is_empty();
        let clear_w = if has_rails { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_rails { 4.0 } else { 0.0 }).max(96.0);
        let label = if has_rails {
            format!("Guide Rails ({})##bp-rails", self.guide_rails.len())
        } else {
            "Guide Rails##bp-rails".to_string()
        };
        if selector_pill(ui, &label, has_rails, active, [pill_w, 22.0]) {
            self.set_selection_mode(GuiBoundaryPatchSelectionMode::GuideRails, actions);
        }
        if has_rails {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##bp-rails-clear", [clear_w, 22.0]) {
                self.guide_rails.clear();
            }
            for idx in 0..self.guide_rails.len() {
                ui.text(format!("{}. Edge", idx + 1));
            }
        }
    }

    pub(super) fn draw_output_and_status(&self, ui: &Ui, state: &GuiCommandState) {
        crate::theme::property_label(ui, "Body Name");
        let next_idx = state
            .browser
            .as_ref()
            .map_or(1, |b| b.surface_bodies.len() + 1);
        ui.align_text_to_frame_padding();
        ui.text(format!("Srf{next_idx}"));
        if let Some(preview) = &state.boundary_patch_preview
            && let GuiBoundaryPatchPreview::Failed(msg) = &preview.result
        {
            ui.text_colored(crate::theme::error_accent(ui), msg);
        }
    }

    pub(super) fn draw_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let ready_request = self.request(state).filter(|req| {
            GuiShellAction::BoundaryPatch(GuiBoundaryPatchAction::Commit(Box::new(req.clone())))
                .dispatchable(state)
        });
        let can_commit = ready_request.is_some();
        if crate::theme::property_accept(ui, can_commit)
            && let Some(req) = ready_request.clone()
        {
            self.submitted = Some(req.clone());
            self.reopen_after_commit = false;
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::Commit(Box::new(req)),
            ));
        }
        ui.same_line_with_spacing(0.0, 6.0);
        let _apply_disabled = ui.begin_disabled(!can_commit);
        if widget_abi::button(ui, "Apply", [58.0, 0.0])
            && let Some(req) = ready_request
        {
            self.submitted = Some(req.clone());
            self.reopen_after_commit = true;
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::Commit(Box::new(req)),
            ));
        }
        drop(_apply_disabled);
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
