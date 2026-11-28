// SPDX-License-Identifier: GPL-2.0-only
//! Behavior controls and Angle A options menu for the Revolution Properties panel.
use super::{DIRECTIONS, RevolveDialog};
use crate::{
    GuiRevolveExtent, GuiRevolvePreview,
    feature_properties::draw_error_badge,
    icons::{GuiRibbonIcons, IconGlyph},
    parameters::GuiParameterEditor,
    widget_abi,
};
use imgui::Ui;
use oblikovati_model::feature::sketch_revolve::SketchRevolveDirection;

const ANGLE_MENU: &str = "revolve-angle-a-options";
const OPTIONS_BUTTON_WIDTH: f32 = 24.;

impl RevolveDialog {
    pub(super) fn draw_behavior(
        &mut self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        result: Option<&GuiRevolvePreview>,
        document: u64,
        parameters: &mut GuiParameterEditor,
    ) {
        if !ui.collapsing_header("Behavior", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        self.draw_extent_buttons(ui);
        if self.extent == GuiRevolveExtent::ToReference {
            crate::theme::property_label(ui, "From");
            ui.text("1 Sketch Plane");
            crate::theme::property_label(ui, "To");
            ui.text("Select Face");
            return;
        }
        self.draw_direction_selector(ui, icons);
        if self.extent == GuiRevolveExtent::Full {
            crate::theme::property_label(ui, "Angle A");
            ui.text("(360 deg)");
            return;
        }
        self.draw_angle_fields(ui, icons, result, document, parameters);
    }

    fn draw_extent_buttons(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Extent");
        if widget_abi::button(ui, "Full", [0.; 2]) {
            self.extent = GuiRevolveExtent::Full;
            self.angle = "360 deg".into();
            self.presets.clear_selection();
        }
        ui.same_line();
        if widget_abi::button(ui, "Angle", [0.; 2]) {
            self.extent = GuiRevolveExtent::Angle;
            if self.angle == "360 deg" {
                self.angle = "90 deg".into();
            }
            self.presets.clear_selection();
        }
        ui.same_line();
        if widget_abi::button(ui, "To", [0.; 2]) {
            self.extent = GuiRevolveExtent::ToReference;
            self.presets.clear_selection();
        }
    }

    fn draw_direction_selector(&mut self, ui: &Ui, icons: &GuiRibbonIcons) {
        self.direction_style
            .draw_style_menu(ui, "Direction", "##revolve-direction-style");
        if let Some(chosen) = self.direction_style.choose(
            ui,
            icons,
            "##revolve-direction",
            self.direction,
            &DIRECTIONS,
            |_| true,
        ) {
            self.direction = chosen;
            if chosen == SketchRevolveDirection::Asymmetric {
                self.extent = GuiRevolveExtent::Angle;
                if self.second_angle.trim().is_empty() {
                    self.second_angle = "45,00 deg".into();
                }
            }
            self.presets.clear_selection();
        }
    }

    fn draw_angle_fields(
        &mut self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        result: Option<&GuiRevolvePreview>,
        document: u64,
        parameters: &mut GuiParameterEditor,
    ) {
        crate::theme::property_label(ui, "Angle A");
        let spacing = ui.clone_style().item_spacing[0];
        let failed = matches!(result, Some(GuiRevolvePreview::Failed(_)));
        let badge_width = if failed { 26. } else { 0. };
        ui.set_next_item_width(
            (ui.content_region_avail()[0] - OPTIONS_BUTTON_WIDTH - spacing - badge_width).max(60.),
        );
        if ui.input_text("##revolve-angle-a", &mut self.angle).build() {
            self.presets.clear_selection();
        }
        if let Some(GuiRevolvePreview::Failed(message)) = result {
            draw_error_badge(ui, message);
        }
        ui.same_line();
        if icons.navigation_button(
            ui,
            "Angle A Options",
            IconGlyph::Parameters,
            [OPTIONS_BUTTON_WIDTH, ui.frame_height()],
        ) {
            ui.open_popup(ANGLE_MENU);
        }
        if let Some(_popup) = ui.begin_popup(ANGLE_MENU)
            && ui.menu_item("Parameters")
        {
            parameters.open(Some(document));
        }
        if let Some(GuiRevolvePreview::Failed(message)) = result
            && message.contains("Out of Range")
        {
            ui.text("Out of Range");
        }
        if self.direction == SketchRevolveDirection::Asymmetric {
            crate::theme::property_label(ui, "Angle B");
            ui.set_next_item_width((ui.content_region_avail()[0] - 28.).max(60.));
            if ui
                .input_text("##revolve-angle-b", &mut self.second_angle)
                .build()
            {
                self.presets.clear_selection();
            }
        }
    }
}
