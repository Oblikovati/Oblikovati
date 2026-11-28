// SPDX-License-Identifier: GPL-2.0-only
//! Tabbed Stitch and Analyze controls, edge-condition legends, and Apply/Done footer (#3918).
use super::StitchDialog;
use super::types::{GuiStitchAction, GuiStitchFreeEdge, GuiStitchPreview};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

impl StitchDialog {
    pub(super) fn draw_tabs(&mut self, ui: &Ui, state: &GuiCommandState) {
        if let Some(_tabs) = ui.tab_bar("##stitch-tabs") {
            if let Some(_stitch_tab) = ui.tab_item("Stitch") {
                self.draw_stitch_tab(ui, state);
            }
            if let Some(_analyze_tab) = ui.tab_item("Analyze") {
                self.draw_analyze_tab(ui, state);
            }
        }
    }

    fn draw_stitch_tab(&mut self, ui: &Ui, state: &GuiCommandState) {
        crate::theme::property_label(ui, "Surfaces");
        let count = self.surfaces.len();
        let has_surfaces = count > 0;
        let count_label = format!("{count} Selected##stitch-surfaces-pill");
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_surfaces { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_surfaces { 4.0 } else { 0.0 }).max(96.0);
        let pill_color = if count >= 2 {
            [0.20, 0.44, 0.32, 1.0]
        } else {
            [0.24, 0.44, 0.78, 1.0]
        };
        let _bg = widget_abi::push_style_color(ui, imgui::StyleColor::Button, pill_color);
        let _ = widget_abi::button(ui, &count_label, [pill_w, 22.0]);
        drop(_bg);
        if has_surfaces {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##stitch-surfaces-clear", [clear_w, 22.0]) {
                self.surfaces.clear();
                self.surface_names.clear();
                self.applied_free_edges.clear();
                self.applied_stitched_count = None;
            }
        }
        if !self.surface_names.is_empty() {
            ui.text_disabled(self.surface_names.join(", "));
        }

        ui.spacing();
        ui.checkbox("Maintain as surface", &mut self.maintain_as_surface);

        ui.spacing();
        crate::theme::property_label(ui, "Maximum Tolerance");
        let tol_valid = self.parsed_tolerance_mm().is_some();
        let field_w = (ui.content_region_avail()[0] - if tol_valid { 0.0 } else { 20.0 }).max(80.0);
        let border_color = (!tol_valid).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Border,
                crate::theme::error_accent(ui),
            )
        });
        let border_w =
            (!tol_valid).then(|| ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.5)));
        ui.set_next_item_width(field_w);
        ui.input_text("##stitch-tolerance", &mut self.tolerance_text)
            .build();
        drop(border_w);
        drop(border_color);
        if !tol_valid {
            crate::feature_properties::draw_error_badge(ui, "Invalid maximum tolerance");
        }

        ui.spacing();
        ui.separator();
        self.draw_remaining_free_edges(ui, state);

        if let Some(preview) = &state.stitch_preview
            && let GuiStitchPreview::Failed(msg) = &preview.result
        {
            ui.spacing();
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
    }

    fn draw_remaining_free_edges(&self, ui: &Ui, state: &GuiCommandState) {
        let active_edges: &[GuiStitchFreeEdge] = match state.stitch_preview.as_ref() {
            Some(preview) => match &preview.result {
                GuiStitchPreview::Ready { free_edges, .. } => free_edges.as_slice(),
                _ => self.applied_free_edges.as_slice(),
            },
            None => self.applied_free_edges.as_slice(),
        };
        ui.text(format!("Remaining Free Edges ({})", active_edges.len()));
        ui.child_window("##stitch-free-edges")
            .size([0.0, 128.0])
            .border(true)
            .build(|| {
                if active_edges.is_empty() {
                    ui.text_disabled("No remaining free edges");
                    return;
                }
                for edge in active_edges {
                    let row = match edge.max_gap_mm {
                        Some(gap) => format!(
                            "{}  ({:.3} mm, gap {:.3} mm)",
                            edge.label, edge.length_mm, gap
                        ),
                        None => format!("{}  ({:.3} mm)", edge.label, edge.length_mm),
                    };
                    ui.text(row);
                }
            });
    }

    fn draw_analyze_tab(&mut self, ui: &Ui, state: &GuiCommandState) {
        ui.checkbox("Show Edge Conditions", &mut self.show_edge_conditions);
        if self.show_edge_conditions {
            ui.indent();
            draw_legend_row(ui, [0.15, 0.85, 0.25, 1.0], "Tangent / Stitchable");
            draw_legend_row(ui, [0.18, 0.18, 0.22, 1.0], "Stitched Edges");
            draw_legend_row(ui, [0.90, 0.22, 0.22, 1.0], "Free Edges");
            ui.unindent();
        }
        ui.spacing();
        ui.checkbox("Show Near Tangent", &mut self.show_near_tangent);
        if self.show_near_tangent {
            ui.indent();
            draw_legend_row(ui, [0.95, 0.35, 0.95, 1.0], "Near Tangent");
            ui.unindent();
        }
        if let Some(preview) = &state.stitch_preview
            && let GuiStitchPreview::Ready {
                stitched_edge_count,
                free_edges,
                near_tangent_count,
            } = &preview.result
        {
            ui.spacing();
            ui.separator();
            ui.text(format!("Stitched edges: {stitched_edge_count}"));
            ui.text(format!("Free edges: {}", free_edges.len()));
            ui.text(format!("Near tangent edges: {near_tangent_count}"));
        }
    }

    pub(super) fn draw_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let ready_request = (self.surfaces.len() >= 2 && self.parsed_tolerance_mm().is_some())
            .then(|| self.request(state))
            .flatten()
            .filter(|req| {
                GuiShellAction::Stitch(GuiStitchAction::Apply(Box::new(req.clone())))
                    .dispatchable(state)
            });
        let _apply_disabled = ui.begin_disabled(ready_request.is_none());
        if widget_abi::button(ui, "Apply", [76.0, 26.0])
            && let Some(req) = ready_request
        {
            self.submitted = Some(req.clone());
            actions.push(GuiShellAction::Stitch(GuiStitchAction::Apply(Box::new(
                req,
            ))));
        }
        drop(_apply_disabled);
        ui.same_line_with_spacing(0.0, 8.0);
        if widget_abi::button(ui, "Done", [76.0, 26.0])
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }
}

fn draw_legend_row(ui: &Ui, color: [f32; 4], label: &str) {
    let cursor = ui.cursor_screen_pos();
    let swatch = 12.0;
    let y_off = ((ui.text_line_height() - swatch) * 0.5).max(0.0);
    let min = [cursor[0], cursor[1] + y_off];
    let max = [cursor[0] + swatch, cursor[1] + y_off + swatch];
    let draw_list = ui.get_window_draw_list();
    draw_list.add_rect(min, max, color).filled(true).build();
    draw_list
        .add_rect(min, max, [0.45, 0.45, 0.48, 1.0])
        .filled(false)
        .build();
    widget_abi::dummy(ui, [swatch, ui.text_line_height()]);
    ui.same_line_with_spacing(0.0, 6.0);
    ui.text(label);
}
