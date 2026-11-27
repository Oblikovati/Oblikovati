// SPDX-License-Identifier: GPL-2.0-only
//! Icon-only controls for the viewport navigation bar.
use super::*;
use crate::icons::{GuiRibbonIcons, IconGlyph};

impl NavigationBar {
    /// The bar's buttons; example: inside its window.
    pub(in super::super) fn buttons(
        &mut self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        state: &GuiCommandState,
        extent: [u32; 2],
        actions: &mut Vec<GuiShellAction>,
    ) {
        let _spacing = ui.push_style_var(imgui::StyleVar::ItemSpacing([0., 4.]));
        if icons.navigation_button(ui, "Navigation Wheel", IconGlyph::NavWheel, BUTTON) {
            self.wheel.request();
        }
        if self.mode_button(ui, icons, "Pan", IconGlyph::NavPan, GuiNavigationMode::Pan) {
            // Pan's own control ends a rotation, as the recording shows; it starts nothing then.
            if self.is_rotating() {
                self.end(actions);
            } else {
                self.start(GuiNavigationMode::Pan, actions);
            }
        }
        {
            let _tight = ui.push_style_var(imgui::StyleVar::ItemSpacing([0., 0.]));
            if self.mode_button(
                ui,
                icons,
                "Zoom",
                IconGlyph::NavZoom,
                GuiNavigationMode::Zoom,
            ) {
                self.run_zoom_action(self.zoom_action, state, extent, actions);
            }
            if menu_button(ui, "##zoom-actions", "Zoom options") {
                ui.open_popup(ZOOM_MENU);
            }
        }
        {
            let _tight = ui.push_style_var(imgui::StyleVar::ItemSpacing([0., 0.]));
            if self.mode_button(
                ui,
                icons,
                "Orbit",
                IconGlyph::NavOrbit,
                GuiNavigationMode::Orbit,
            ) {
                self.run_orbit_action(actions);
            }
            if menu_button(ui, "##orbit-actions", "Orbit options") {
                ui.open_popup(ORBIT_MENU);
            }
        }
        self.zoom_menu(ui, state, extent, actions);
        self.orbit_menu(ui, actions);
    }
    fn mode_button(
        &self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        label: &str,
        glyph: IconGlyph,
        mode: GuiNavigationMode,
    ) -> bool {
        let active = ui.style_color(StyleColor::ButtonActive);
        let _lit =
            (self.mode == Some(mode)).then(|| ui.push_style_color(StyleColor::Button, active));
        icons.navigation_button(ui, label, glyph, BUTTON)
    }
}

fn menu_button(ui: &Ui, id: &str, tooltip: &str) -> bool {
    let _bg = ui.push_style_color(StyleColor::Button, [0.; 4]);
    let _bw = ui.push_style_var(imgui::StyleVar::FrameBorderSize(0.));
    let pressed = crate::widget_abi::button(ui, id, [BUTTON[0], 9.]);
    let [left, top] = ui.item_rect_min();
    let [right, bottom] = ui.item_rect_max();
    let [x, y] = [
        ((left + right) * 0.5).floor(),
        ((top + bottom) * 0.5).floor(),
    ];
    let ink = if ui.is_item_hovered() {
        ui.style_color(StyleColor::Text)
    } else {
        crate::theme::text_muted(ui)
    };
    ui.get_window_draw_list()
        .add_triangle([x - 3., y - 1.], [x + 3., y - 1.], [x, y + 2.], ink)
        .filled(true)
        .build();
    if ui.is_item_hovered() {
        ui.tooltip_text(tooltip);
    }
    pressed
}
