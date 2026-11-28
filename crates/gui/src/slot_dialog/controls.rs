// SPDX-License-Identifier: GPL-2.0-only
//! Shape, Type, Seat, Center Marks, End Mill, Behavior, and Dimension controls for Slot Properties.
use super::SlotDialog;
use super::catalog::{
    CLEARANCE_FASTENER_TYPES, CLEARANCE_SIZES, CLEARANCE_STANDARDS, default_fastener, fit_label,
    parse_length_mm, surface_label, termination_label,
};
use crate::feature_properties::draw_pick_pill;
use crate::widget_abi;
use crate::{GuiHoleSeatKind, GuiHoleTermination};
use imgui::Ui;
use oblikovati_model::feature::{
    hole::{HoleFastenerFit, HoleKind},
    slot::{SlotCenterMarks, SlotShape},
};

impl SlotDialog {
    pub(super) fn draw_input_geometry(&mut self, ui: &Ui) {
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
        let has_pos = self.face.is_some();
        crate::theme::property_label(ui, "Positions");
        draw_pick_pill(
            ui,
            if has_pos {
                "1 Position"
            } else {
                "Select positions"
            },
            has_pos,
        );
    }

    pub(super) fn draw_shape_type_and_seat(&mut self, ui: &Ui, invalid: bool) {
        let _pad = ui.push_style_var(imgui::StyleVar::FramePadding([2.0, 2.0]));
        let row_w = (ui.content_region_avail()[0] - 9.0).max(160.0);

        crate::theme::property_label(ui, "Shape");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
        if toggle_button(
            ui,
            "Rounded",
            self.shape == SlotShape::Rounded,
            false,
            [half, 20.0],
        ) {
            self.shape = SlotShape::Rounded;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Rect",
            self.shape == SlotShape::Rectangular,
            false,
            [half, 20.0],
        ) {
            self.shape = SlotShape::Rectangular;
        }

        crate::theme::property_label(ui, "Hole Type");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
        if toggle_button(
            ui,
            "Simple",
            self.kind == HoleKind::Simple,
            false,
            [half, 20.0],
        ) && self.kind != HoleKind::Simple
        {
            self.kind = HoleKind::Simple;
            self.fastener = None;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Clearance",
            self.kind == HoleKind::Clearance,
            false,
            [half, 20.0],
        ) && self.kind != HoleKind::Clearance
        {
            self.kind = HoleKind::Clearance;
            let spec = self.fastener.get_or_insert_with(default_fastener).clone();
            self.apply_clearance_dimensions(&spec);
        }

        ui.text("Seat");
        let seat_bad = invalid && self.seat != GuiHoleSeatKind::None;
        for (idx, (seat, label, share)) in [
            (GuiHoleSeatKind::None, "None", 0.20_f32),
            (GuiHoleSeatKind::Counterbore, "C'bore", 0.25),
            (GuiHoleSeatKind::Spotface, "Spotface", 0.31),
            (GuiHoleSeatKind::Countersink, "C'sink", 0.24),
        ]
        .into_iter()
        .enumerate()
        {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 3.0);
            }
            let selected = self.seat == seat;
            if toggle_button(
                ui,
                label,
                selected,
                selected && seat_bad,
                [row_w * share, 20.0],
            ) && !selected
            {
                self.set_seat_kind(seat);
            }
        }
    }

    fn set_seat_kind(&mut self, seat: GuiHoleSeatKind) {
        self.seat = seat;
        let w = parse_length_mm(&self.width).unwrap_or(2.0);
        match seat {
            GuiHoleSeatKind::None => {}
            GuiHoleSeatKind::Counterbore => {
                if parse_length_mm(&self.seat_width).is_none_or(|sw| sw <= w) {
                    self.seat_width = format!("{} mm", w + 2.0);
                }
                if self.seat_depth.trim().is_empty() {
                    self.seat_depth = "2 mm".into();
                }
            }
            GuiHoleSeatKind::Spotface => {
                if parse_length_mm(&self.seat_width).is_none_or(|sw| sw <= w) {
                    self.seat_width = format!("{} mm", w + 2.0);
                }
                if self.seat_depth.trim().is_empty() {
                    self.seat_depth = "1 mm".into();
                }
            }
            GuiHoleSeatKind::Countersink => {
                if parse_length_mm(&self.seat_width).is_none_or(|sw| sw <= w) {
                    self.seat_width = format!("{} mm", w + 2.0);
                }
                self.seat_angle = "90 deg".into();
            }
        }
    }

    pub(super) fn draw_clearance_controls(&mut self, ui: &Ui) {
        let mut spec = self.fastener.clone().unwrap_or_else(default_fastener);
        let mut changed = false;
        changed |= draw_combo_row(
            ui,
            "Standard",
            "##slot-std",
            &mut spec.standard,
            CLEARANCE_STANDARDS,
        );
        changed |= draw_combo_row(
            ui,
            "Fastener Type",
            "##slot-fastener",
            &mut spec.fastener_type,
            CLEARANCE_FASTENER_TYPES,
        );
        changed |= draw_combo_row(ui, "Size", "##slot-size", &mut spec.size, CLEARANCE_SIZES);
        let mut fit_text = fit_label(spec.fit).to_owned();
        if draw_combo_row(
            ui,
            "Fit",
            "##slot-fit",
            &mut fit_text,
            &["Close", "Normal", "Loose"],
        ) {
            spec.fit = match fit_text.as_str() {
                "Close" => HoleFastenerFit::Close,
                "Loose" => HoleFastenerFit::Loose,
                _ => HoleFastenerFit::Normal,
            };
            changed = true;
        }
        if changed {
            self.apply_clearance_dimensions(&spec);
        }
        self.fastener = Some(spec);
    }

    pub(super) fn draw_behavior(&mut self, ui: &Ui) {
        let _pad = ui.push_style_var(imgui::StyleVar::FramePadding([2.0, 2.0]));
        let mut term_text = termination_label(self.termination).to_owned();
        if draw_combo_row(
            ui,
            "Termination",
            "##slot-term",
            &mut term_text,
            &["Distance", "Through All", "To Surface"],
        ) {
            self.termination = match term_text.as_str() {
                "Distance" => GuiHoleTermination::Distance,
                "To Surface" => GuiHoleTermination::ToSurface,
                _ => GuiHoleTermination::ThroughAll,
            };
        }
        if self.termination == GuiHoleTermination::ToSurface {
            crate::theme::property_label(ui, "Surface");
            let pill = self
                .termination_surface
                .as_ref()
                .map(surface_label)
                .unwrap_or("Select surface");
            draw_pick_pill(ui, pill, self.termination_surface.is_some());
        }

        crate::theme::property_label(ui, "End Mill");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
        if toggle_button(ui, "Flat", !self.end_mill_pointed, false, [half, 20.0]) {
            self.end_mill_pointed = false;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(ui, "Point", self.end_mill_pointed, false, [half, 20.0]) {
            self.end_mill_pointed = true;
        }

        let row_w = (ui.content_region_avail()[0] - 9.0).max(160.0);
        ui.text("Center Marks");
        for (idx, (mode, label, share)) in [
            (SlotCenterMarks::None, "Off", 0.22_f32),
            (SlotCenterMarks::Center, "Mid", 0.24),
            (SlotCenterMarks::EndCenters, "Ends", 0.27),
            (SlotCenterMarks::All, "All", 0.27),
        ]
        .into_iter()
        .enumerate()
        {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 3.0);
            }
            if toggle_button(
                ui,
                label,
                self.center_marks == mode,
                false,
                [row_w * share, 20.0],
            ) {
                self.center_marks = mode;
            }
        }

        ui.checkbox("iMate", &mut self.imate);
        ui.same_line_with_spacing(0.0, 12.0);
        ui.checkbox("Extend Start", &mut self.extend_start);
    }

    pub(super) fn draw_dimensions(&mut self, ui: &Ui, local_error: Option<&str>) -> bool {
        let mut enter = false;
        let bad_len = local_error.is_some_and(|e| e.contains("length"));
        let bad_wid = local_error.is_some_and(|e| e.contains("width"));
        let bad_seat = local_error.is_some_and(|e| e.contains("Seat") || e.contains("Countersink"));
        enter |= numeric_row(ui, "Length", &mut self.length, bad_len);
        enter |= numeric_row(ui, "Width", &mut self.width, bad_wid);
        enter |= numeric_row(
            ui,
            "Rotation",
            &mut self.rotation,
            local_error.is_some_and(|e| e.contains("rotation")),
        );
        if self.termination == GuiHoleTermination::Distance {
            enter |= numeric_row(
                ui,
                "Depth",
                &mut self.depth,
                local_error.is_some_and(|e| e.contains("depth")),
            );
        }
        match self.seat {
            GuiHoleSeatKind::None => {}
            GuiHoleSeatKind::Counterbore => {
                enter |= numeric_row(ui, "Counterbore", &mut self.seat_width, bad_seat);
                enter |= numeric_row(ui, "Seat Depth", &mut self.seat_depth, bad_seat);
            }
            GuiHoleSeatKind::Spotface => {
                enter |= numeric_row(ui, "Spotface", &mut self.seat_width, bad_seat);
                enter |= numeric_row(ui, "Seat Depth", &mut self.seat_depth, bad_seat);
            }
            GuiHoleSeatKind::Countersink => {
                enter |= numeric_row(ui, "Countersink", &mut self.seat_width, bad_seat);
                enter |= numeric_row(ui, "C'sink Angle", &mut self.seat_angle, bad_seat);
            }
        }
        if self.end_mill_pointed {
            enter |= numeric_row(
                ui,
                "Point",
                &mut self.end_mill_angle,
                local_error.is_some_and(|e| e.contains("Point")),
            );
        }
        enter
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &crate::GuiCommandState,
        enter_commit: bool,
        popup_active: bool,
        actions: &mut Vec<crate::GuiShellAction>,
    ) {
        let local_ok = super::catalog::validate_fields(self).is_none();
        let ready_request = local_ok
            .then(|| self.request(state))
            .flatten()
            .filter(|req| {
                crate::GuiShellAction::Slot(super::types::GuiSlotAction::Commit(Box::new(
                    req.clone(),
                )))
                .dispatchable(state)
            });
        let can_ok = ready_request.is_some();
        let ok_clicked = crate::theme::property_accept(ui, can_ok);
        if (ok_clicked || (enter_commit && can_ok))
            && let Some(req) = ready_request.clone()
        {
            self.submitted = Some(req.clone());
            self.reopen_after_commit = false;
            actions.push(crate::GuiShellAction::Slot(
                super::types::GuiSlotAction::Commit(Box::new(req)),
            ));
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (!popup_active
                && ui.is_window_focused()
                && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
            return;
        }
        let plus_w = 28.0;
        let right = ui.window_content_region_max()[0] - plus_w;
        if right > ui.cursor_pos()[0] + 6.0 {
            ui.same_line_with_spacing(0.0, 0.0);
            ui.set_cursor_pos([right, ui.cursor_pos()[1]]);
        } else {
            ui.same_line_with_spacing(0.0, 6.0);
        }
        let _disabled = ui.begin_disabled(!can_ok);
        if widget_abi::button(ui, "+", [plus_w, 0.0])
            && let Some(req) = ready_request
        {
            self.submitted = Some(req.clone());
            self.reopen_after_commit = true;
            actions.push(crate::GuiShellAction::Slot(
                super::types::GuiSlotAction::Commit(Box::new(req)),
            ));
        }
    }
}

fn toggle_button(ui: &Ui, label: &str, selected: bool, invalid: bool, size: [f32; 2]) -> bool {
    let color = if invalid {
        Some(widget_abi::push_style_color(
            ui,
            imgui::StyleColor::Button,
            [0.72, 0.20, 0.20, 1.0],
        ))
    } else if selected {
        Some(widget_abi::push_style_color(
            ui,
            imgui::StyleColor::Button,
            [0.26, 0.36, 0.68, 1.0],
        ))
    } else {
        None
    };
    let clicked = widget_abi::button(ui, label, size);
    drop(color);
    clicked
}

fn draw_combo_row(ui: &Ui, label: &str, id: &str, current: &mut String, items: &[&str]) -> bool {
    crate::theme::property_label(ui, label);
    let mut changed = false;
    if let Some(_combo) = ui.begin_combo(id, current.as_str()) {
        for &item in items {
            if widget_abi::selectable(ui, item, current == item, false) {
                *current = item.to_owned();
                changed = true;
            }
        }
    }
    changed
}

fn numeric_row(ui: &Ui, label: &str, value: &mut String, invalid: bool) -> bool {
    crate::theme::property_label(ui, label);
    let _id = ui.push_id(label);
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.0));
    let _border = invalid.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Border, [0.88, 0.25, 0.25, 1.0])
    });
    ui.input_text("##value", value)
        .enter_returns_true(true)
        .build()
}
