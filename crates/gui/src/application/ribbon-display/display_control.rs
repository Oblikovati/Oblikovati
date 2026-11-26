// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};

pub(crate) const DENSITY_CONTROL_WIDTH: f32 = 52.;
const DENSITY_BUTTON_SIZE: [f32; 2] = [24., 22.];

impl RibbonDisplayState {
    /// Draw the display control and its menu, for example at `[900., 35.]`.
    pub(crate) fn control(
        &mut self,
        ui: &Ui,
        position: [f32; 2],
        icons: &GuiRibbonIcons,
        tips: &crate::ribbon_key_tips::RibbonKeyTips,
        open_menu: bool,
    ) {
        crate::widget_abi::cursor_position(ui, position);
        let clicked = crate::widget_abi::icon_button(
            ui,
            "Ribbon Display",
            crate::widget_abi::IconButtonPresentation {
                texture: icons.key(IconGlyph::RibbonDensity, IconSize::Small),
                size: DENSITY_BUTTON_SIZE,
                icon: [16.; 2],
                large: false,
                show_caption: false,
            },
        );
        let activate = self.control_press.button(ui, (), clicked);
        if tips.tabs() && tips.matching("X2") {
            crate::ribbon_key_tips::badge(ui, "X2");
        }
        if activate {
            self.advance();
        }
        if ui.is_item_hovered() {
            ui.tooltip_text(self.control_tooltip());
        }
        ui.same_line_with_spacing(0., 0.);
        self.display_menu_button(ui, tips, open_menu);
    }

    fn display_menu_button(
        &mut self,
        ui: &Ui,
        tips: &crate::ribbon_key_tips::RibbonKeyTips,
        open_menu: bool,
    ) {
        let clicked = crate::widget_abi::button(ui, "##ribbon-display-menu", [20., 22.]);
        let [left, top] = ui.item_rect_min();
        let [right, bottom] = ui.item_rect_max();
        let center = [(left + right) / 2., (top + bottom) / 2.];
        ui.get_window_draw_list()
            .add_triangle(
                [center[0] - 4., center[1] - 2.],
                [center[0] + 4., center[1] - 2.],
                [center[0], center[1] + 3.],
                ui.style_color(imgui::StyleColor::Text),
            )
            .filled(true)
            .build();
        if ui.is_item_hovered() {
            ui.tooltip_text("Ribbon Display Options");
        }
        let activate = self.menu_press.button(ui, (), clicked);
        if tips.tabs() && tips.matching("X3") {
            crate::ribbon_key_tips::badge(ui, "X3");
        }
        if activate || open_menu {
            ui.open_popup("##ribbon-display-menu");
        }
        self.keep_temporary_on_click =
            ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Left);
        if let Some(_menu) = ui.begin_popup("##ribbon-display-menu") {
            self.menu(ui);
        } else {
            self.display_menu_cursor = None;
        }
    }
}
