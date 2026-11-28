// SPDX-License-Identifier: GPL-2.0-only
//! Type, Seat, Fastener, Thread, Behavior, and Dimension controls for Hole Properties.
use super::HoleDialog;
use super::catalog::{
    CLEARANCE_FASTENER_TYPES, CLEARANCE_SIZES, CLEARANCE_STANDARDS, TAPER_SIZES,
    TAPER_THREAD_TYPES, TAPPED_SIZES, TAPPED_THREAD_TYPES, default_fastener, default_taper_thread,
    default_tapped_thread, fit_label, parse_length_mm, surface_label, termination_label,
};
use super::types::{GuiHoleSeatKind, GuiHoleTermination};
use crate::feature_properties::draw_pick_pill;
use crate::widget_abi;
use imgui::Ui;
use oblikovati_model::feature::hole::{HoleFastenerFit, HoleKind};

impl HoleDialog {
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

    pub(super) fn draw_type_and_seat(&mut self, ui: &Ui, invalid: bool) {
        let _pad = ui.push_style_var(imgui::StyleVar::FramePadding([2.0, 2.0]));
        let row_w = (ui.content_region_avail()[0] - 9.0).max(160.0);
        ui.text("Type");
        for (idx, (kind, label, share)) in [
            (HoleKind::Simple, "Simple", 0.23_f32),
            (HoleKind::Clearance, "Clearance", 0.32),
            (HoleKind::Tapped, "Tapped", 0.24),
            (HoleKind::TaperTapped, "Taper", 0.21),
        ]
        .into_iter()
        .enumerate()
        {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 3.0);
            }
            let selected = self.kind == kind;
            if toggle_button(ui, label, selected, false, [row_w * share, 22.0]) && !selected {
                self.set_hole_kind(kind);
            }
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
                [row_w * share, 22.0],
            ) && !selected
            {
                self.set_seat_kind(seat);
            }
        }
    }

    fn set_hole_kind(&mut self, kind: HoleKind) {
        self.kind = kind;
        match kind {
            HoleKind::Simple => {
                self.fastener = None;
                self.thread = None;
            }
            HoleKind::Clearance => {
                self.thread = None;
                let spec = self.fastener.get_or_insert_with(default_fastener).clone();
                self.apply_clearance_dimensions(&spec);
            }
            HoleKind::Tapped => {
                self.fastener = None;
                let spec = self.thread.get_or_insert_with(default_tapped_thread);
                if !TAPPED_THREAD_TYPES.contains(&spec.profile.as_str()) {
                    *spec = default_tapped_thread();
                }
            }
            HoleKind::TaperTapped => {
                self.fastener = None;
                self.thread = Some(default_taper_thread());
            }
        }
    }

    fn set_seat_kind(&mut self, seat: GuiHoleSeatKind) {
        self.seat = seat;
        match seat {
            GuiHoleSeatKind::None => {}
            GuiHoleSeatKind::Counterbore => {
                if self.seat_diameter.trim().is_empty() {
                    self.seat_diameter = self.diameter.clone();
                }
                if self.seat_depth.trim().is_empty() {
                    self.seat_depth = "2 mm".into();
                }
            }
            GuiHoleSeatKind::Spotface => {
                if parse_length_mm(&self.seat_diameter)
                    .zip(parse_length_mm(&self.diameter))
                    .is_none_or(|(sd, d)| sd <= d)
                {
                    let d = parse_length_mm(&self.diameter).unwrap_or(4.0);
                    self.seat_diameter = format!("{} mm", d + 2.0);
                }
                if self.seat_depth.trim().is_empty() {
                    self.seat_depth = "1 mm".into();
                }
            }
            GuiHoleSeatKind::Countersink => {
                if parse_length_mm(&self.seat_diameter)
                    .zip(parse_length_mm(&self.diameter))
                    .is_none_or(|(sd, d)| sd <= d)
                {
                    let d = parse_length_mm(&self.diameter).unwrap_or(4.0);
                    self.seat_diameter = format!("{} mm", d + 2.0);
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
            "##hole-std",
            &mut spec.standard,
            CLEARANCE_STANDARDS,
        );
        changed |= draw_combo_row(
            ui,
            "Fastener Type",
            "##hole-fastener",
            &mut spec.fastener_type,
            CLEARANCE_FASTENER_TYPES,
        );
        changed |= draw_combo_row(ui, "Size", "##hole-size", &mut spec.size, CLEARANCE_SIZES);
        let mut fit_text = fit_label(spec.fit).to_owned();
        if draw_combo_row(
            ui,
            "Fit",
            "##hole-fit",
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

    pub(super) fn draw_thread_controls(&mut self, ui: &Ui) {
        let is_taper = self.kind == HoleKind::TaperTapped;
        let mut spec = self.thread.clone().unwrap_or_else(|| {
            if is_taper {
                default_taper_thread()
            } else {
                default_tapped_thread()
            }
        });
        let types = if is_taper {
            TAPER_THREAD_TYPES
        } else {
            TAPPED_THREAD_TYPES
        };
        draw_combo_row(
            ui,
            "Thread Type",
            "##hole-thread-type",
            &mut spec.profile,
            types,
        );
        let size_rows = if is_taper { TAPER_SIZES } else { TAPPED_SIZES };
        let size_labels: Vec<&str> = size_rows.iter().map(|(s, _, _)| *s).collect();
        let mut current_size = size_rows
            .iter()
            .find(|(_, desig, _)| *desig == spec.designation)
            .map_or(size_rows[0].0, |(s, _, _)| *s)
            .to_owned();
        if draw_combo_row(
            ui,
            "Size",
            "##hole-thread-size",
            &mut current_size,
            &size_labels,
        ) && let Some((_, designation, tap_diam)) =
            size_rows.iter().find(|(s, _, _)| *s == current_size)
        {
            spec.designation = (*designation).into();
            self.diameter = (*tap_diam).into();
        }
        crate::theme::property_label(ui, "Designation");
        ui.text(&spec.designation);
        if !is_taper {
            ui.same_line_with_spacing(0.0, 8.0);
            ui.checkbox("Full Depth", &mut spec.full_depth);
            crate::theme::property_label(ui, "Thread Dir");
            let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
            if toggle_button(ui, "Right", !spec.left_hand, false, [half, 20.0]) {
                spec.left_hand = false;
            }
            ui.same_line_with_spacing(0.0, 4.0);
            if toggle_button(ui, "Left", spec.left_hand, false, [half, 20.0]) {
                spec.left_hand = true;
            }
        }
        self.thread = Some(spec);
    }

    pub(super) fn draw_behavior(&mut self, ui: &Ui) {
        let mut term_text = termination_label(self.termination).to_owned();
        if draw_combo_row(
            ui,
            "Termination",
            "##hole-term",
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
        crate::theme::property_label(ui, "Direction");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
        if toggle_button(ui, "Default", !self.direction_flipped, false, [half, 20.0]) {
            self.direction_flipped = false;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(ui, "Flipped", self.direction_flipped, false, [half, 20.0]) {
            self.direction_flipped = true;
        }
        if self.termination == GuiHoleTermination::Distance {
            crate::theme::property_label(ui, "Drill Point");
            let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
            if toggle_button(ui, "Flat", !self.drill_point_angled, false, [half, 20.0]) {
                self.drill_point_angled = false;
            }
            ui.same_line_with_spacing(0.0, 4.0);
            if toggle_button(ui, "Angle", self.drill_point_angled, false, [half, 20.0]) {
                self.drill_point_angled = true;
            }
        }
        ui.checkbox("Extend Start", &mut self.extend_start);
    }

    pub(super) fn draw_dimensions(&mut self, ui: &Ui, local_error: Option<&str>) -> bool {
        let mut enter = false;
        let bad_seat = local_error.is_some_and(|e| e.contains("Seat") || e.contains("Countersink"));
        enter |= numeric_row(
            ui,
            "Diameter",
            &mut self.diameter,
            local_error.is_some_and(|e| e.contains("Hole diameter")),
        );
        if self.termination == GuiHoleTermination::Distance {
            enter |= numeric_row(
                ui,
                "Depth",
                &mut self.depth,
                local_error.is_some_and(|e| e.contains("Hole depth")),
            );
        }
        match self.seat {
            GuiHoleSeatKind::None => {}
            GuiHoleSeatKind::Counterbore | GuiHoleSeatKind::Spotface => {
                enter |= numeric_row(ui, "Seat Diameter", &mut self.seat_diameter, bad_seat);
                enter |= numeric_row(ui, "Seat Depth", &mut self.seat_depth, bad_seat);
            }
            GuiHoleSeatKind::Countersink => {
                enter |= numeric_row(ui, "Seat Diameter", &mut self.seat_diameter, bad_seat);
                enter |= numeric_row(ui, "Seat Angle", &mut self.seat_angle, bad_seat);
            }
        }
        if self.termination == GuiHoleTermination::Distance && self.drill_point_angled {
            enter |= numeric_row(
                ui,
                "Point Angle",
                &mut self.drill_point_angle,
                local_error.is_some_and(|e| e.contains("Drill point")),
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
                crate::GuiShellAction::Hole(super::types::GuiHoleAction::Commit(Box::new(
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
            actions.push(crate::GuiShellAction::Hole(
                super::types::GuiHoleAction::Commit(Box::new(req)),
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
            actions.push(crate::GuiShellAction::Hole(
                super::types::GuiHoleAction::Commit(Box::new(req)),
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
