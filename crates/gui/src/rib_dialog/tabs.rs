// SPDX-License-Identifier: GPL-2.0-only
//! Draft and Boss tabs plus the Hold Thickness cross-section diagram for dockable Rib.
use super::{ERROR_TEXT_RED, GuiRibPreview, RibDialog};
use crate::{GuiCommandState, feature_properties::draw_error_badge, widget_abi};
use imgui::{StyleColor, Ui};
use oblikovati_model::feature::sketch_rib::{RibDirection, RibThickenSide, RibThicknessPlane};

impl RibDialog {
    pub(super) fn draw_shape_tab(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        result: Option<&GuiRibPreview>,
        thickness_invalid: bool,
    ) {
        crate::theme::property_label(ui, "Profile");
        let has_profile = self.sketch.is_some() && !self.profile_curves.is_empty();
        let profile_label = if has_profile {
            "1 Profile"
        } else {
            "Select Profile"
        };
        if widget_abi::button(ui, profile_label, [146., 26.])
            && let Some(candidate) = state
                .rib_sketches
                .iter()
                .find(|item| !item.curves.is_empty())
        {
            self.sketch = Some(candidate.sketch);
            self.profile_curves = candidate.curves.clone();
        }
        if has_profile {
            ui.same_line_with_spacing(0., 6.);
            if widget_abi::button(ui, "Clear##rib-profile", [54., 26.]) {
                self.sketch = None;
                self.profile_curves.clear();
            }
        }
        if state.rib_sketches.len() > 1 {
            crate::theme::property_label(ui, "Sketch");
            ui.set_next_item_width(-1.);
            let active_name = self
                .sketch
                .and_then(|id| state.rib_sketches.iter().find(|s| s.sketch == id))
                .map_or("Choose Sketch", |s| s.name.as_str());
            if let Some(_combo) = ui.begin_combo("##rib-sketch-combo", active_name) {
                for candidate in &state.rib_sketches {
                    if ui
                        .selectable_config(&candidate.name)
                        .selected(self.sketch == Some(candidate.sketch))
                        .build()
                    {
                        self.sketch = Some(candidate.sketch);
                        self.profile_curves = candidate.curves.clone();
                    }
                }
            }
        }

        if let Some(browser) = &state.browser
            && !browser.bodies.is_empty()
        {
            crate::theme::property_label(ui, "Solids");
            let current_solid = self
                .target_body
                .as_deref()
                .and_then(|key| browser.bodies.iter().find(|b| b.key == key))
                .map_or("Select Solid", |b| b.name.as_str());
            ui.set_next_item_width(-1.);
            if let Some(_combo) = ui.begin_combo("##rib-solids", current_solid) {
                for body in &browser.bodies {
                    if ui
                        .selectable_config(&body.name)
                        .selected(self.target_body.as_deref() == Some(body.key.as_str()))
                        .build()
                    {
                        self.target_body = Some(body.key.clone());
                    }
                }
            }
        }

        crate::theme::property_label(ui, "Direction");
        ui.radio_button(
            "Parallel to Sketch Plane",
            &mut self.direction,
            RibDirection::ParallelToSketchPlane,
        );
        ui.radio_button(
            "Normal to Sketch Plane",
            &mut self.direction,
            RibDirection::NormalToSketchPlane,
        );
        if widget_abi::button(ui, "Flip Direction", [0.; 2]) {
            self.flip_direction = !self.flip_direction;
        }

        crate::theme::property_label(ui, "Thickness");
        ui.set_next_item_width(-1.);
        {
            let _numeric = crate::ui_scale::numeric_font(ui);
            let _red =
                thickness_invalid.then(|| ui.push_style_color(StyleColor::Text, ERROR_TEXT_RED));
            ui.input_text("##rib-thickness", &mut self.thickness)
                .build();
        }
        crate::theme::property_label(ui, "Side");
        let side_label = match self.thicken_side {
            RibThickenSide::Symmetric => "Symmetric",
            RibThickenSide::Side1 => "Side 1",
            RibThickenSide::Side2 => "Side 2",
        };
        ui.set_next_item_width(-1.);
        if let Some(_combo) = ui.begin_combo("##rib-side", side_label) {
            for (side, label) in [
                (RibThickenSide::Symmetric, "Symmetric"),
                (RibThickenSide::Side1, "Side 1"),
                (RibThickenSide::Side2, "Side 2"),
            ] {
                if ui
                    .selectable_config(label)
                    .selected(self.thicken_side == side)
                    .build()
                {
                    self.thicken_side = side;
                }
            }
        }

        crate::theme::property_label(ui, "Depth");
        if ui.radio_button_bool("To Next", self.to_next) {
            self.to_next = true;
        }
        ui.same_line_with_spacing(0., 12.);
        if ui.radio_button_bool("Finite", !self.to_next) {
            self.to_next = false;
        }
        if !self.to_next {
            ui.set_next_item_width(-1.);
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.input_text("##rib-depth", &mut self.depth).build();
        }
        ui.checkbox("Extend Profile", &mut self.extend_profile);

        if let Some(GuiRibPreview::Failed(msg)) = result {
            draw_error_badge(ui, msg);
        }
    }

    pub(super) fn draw_draft_tab(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Hold Thickness");
        ui.radio_button(
            "At Top",
            &mut self.thickness_plane,
            RibThicknessPlane::AtTop,
        );
        ui.same_line_with_spacing(0., 16.);
        ui.radio_button(
            "At Root",
            &mut self.thickness_plane,
            RibThicknessPlane::AtRoot,
        );
        crate::theme::property_label(ui, "Draft Angle");
        ui.set_next_item_width(-1.);
        {
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.input_text("##rib-draft-angle", &mut self.draft).build();
        }
        ui.spacing();
        draw_hold_thickness_diagram(ui, self.thickness_plane);
    }

    pub(super) fn draw_boss_tab(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Centers");
        widget_abi::button(ui, "Select Centers", [140., 26.]);
        ui.checkbox("Select All", &mut self.boss_select_all);
        crate::theme::property_label(ui, "Diameter");
        ui.set_next_item_width(-1.);
        {
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.input_text("##rib-boss-diameter", &mut self.boss_diameter)
                .build();
        }
        crate::theme::property_label(ui, "Offset");
        ui.set_next_item_width(-1.);
        {
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.input_text("##rib-boss-offset", &mut self.boss_offset)
                .build();
        }
        crate::theme::property_label(ui, "Draft Angle");
        ui.set_next_item_width(-1.);
        {
            let _numeric = crate::ui_scale::numeric_font(ui);
            ui.input_text("##rib-boss-draft", &mut self.boss_draft)
                .build();
        }
    }
}

fn draw_hold_thickness_diagram(ui: &Ui, mode: RibThicknessPlane) {
    let origin = ui.cursor_screen_pos();
    let width = ui.content_region_avail()[0].max(220.);
    let height = 112.;
    ui.dummy([width, height]);
    let draw = ui.get_window_draw_list();
    let bg = ui.style_color(StyleColor::FrameBg);
    let border = crate::theme::border_subtle(ui);
    let ink = ui.style_color(StyleColor::Text);
    let accent = ui.style_color(StyleColor::CheckMark);
    draw.add_rect(origin, [origin[0] + width, origin[1] + height], bg)
        .filled(true)
        .rounding(4.)
        .build();
    draw.add_rect(origin, [origin[0] + width, origin[1] + height], border)
        .rounding(4.)
        .build();
    let cx = origin[0] + width * 0.5;
    let top_y = origin[1] + 30.;
    let root_y = origin[1] + 84.;
    let (top_half, root_half, dim_y, dim_half, label) = match mode {
        RibThicknessPlane::AtTop => (26., 44., top_y - 12., 26., "T (At Top)"),
        RibThicknessPlane::AtRoot => (18., 36., root_y + 12., 36., "T (At Root)"),
    };
    let poly = [
        [cx - top_half, top_y],
        [cx + top_half, top_y],
        [cx + root_half, root_y],
        [cx - root_half, root_y],
    ];
    let fill = [accent[0], accent[1], accent[2], 0.22];
    draw.add_triangle(poly[0], poly[1], poly[2], fill)
        .filled(true)
        .build();
    draw.add_triangle(poly[0], poly[2], poly[3], fill)
        .filled(true)
        .build();
    draw.add_polyline(poly.to_vec(), ink)
        .filled(false)
        .thickness(1.5)
        .build();
    draw.add_line(poly[3], poly[0], ink).thickness(1.5).build();
    draw.add_line(
        [origin[0] + 20., root_y],
        [origin[0] + width - 20., root_y],
        ink,
    )
    .thickness(1.5)
    .build();
    draw.add_line([cx - dim_half, dim_y], [cx + dim_half, dim_y], accent)
        .thickness(2.0)
        .build();
    draw.add_line(
        [cx - dim_half, dim_y - 4.],
        [cx - dim_half, dim_y + 4.],
        accent,
    )
    .thickness(2.0)
    .build();
    draw.add_line(
        [cx + dim_half, dim_y - 4.],
        [cx + dim_half, dim_y + 4.],
        accent,
    )
    .thickness(2.0)
    .build();
    let text_x = origin[0] + 10.;
    let text_y = if mode == RibThicknessPlane::AtTop {
        origin[1] + 8.
    } else {
        origin[1] + height - 20.
    };
    draw.add_text([text_x, text_y], accent, label);
}
