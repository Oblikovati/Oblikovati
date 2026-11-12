// SPDX-License-Identifier: GPL-2.0-only
//! The direct panel list shares visibility with the context menu but stays open on a change (#3856).
use super::*;
pub(crate) const PANEL_SELECTOR_WIDTH: f32 = 32.;
const SELECTOR_POPUP: &str = "##ribbon-panel-selector-list";
impl RibbonDisplayState {
    /// Toggle panels without reopening the list; example: `display.panel_selector(ui, tab, panels)`.
    pub(crate) fn panel_selector(&mut self, ui: &Ui, tab: RibbonTab, panels: &[RibbonPanel]) {
        self.selector_escape =
            self.selector_open && ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        let temporary_tab =
            self.mode == RibbonDisplay::Tabs && self.temporary == Some(TemporaryPanel::WholeTab);
        if (self.mode != RibbonDisplay::Full && !temporary_tab)
            || self.placement != RibbonPlacement::Top
        {
            self.selector_keyboard_started = false;
            self.selector_open = false;
            self.selector_pointer_inside = false;
            self.selector_trigger = None;
            return;
        }
        let width = ui.window_size()[0];
        let clicked = selector_button(ui, width);
        self.selector_trigger = Some([ui.item_rect_min(), ui.item_rect_max()]);
        if temporary_tab && (clicked || self.selector_pointer_press(ui)) {
            self.keep_temporary_on_click = true;
        }
        let origin = ui.window_pos();
        crate::widget_abi::next_window_position(ui, [origin[0] + width - 180., origin[1] + 60.]);
        self.selector_choices(ui, tab, panels);
    }
    fn selector_choices(&mut self, ui: &Ui, tab: RibbonTab, panels: &[RibbonPanel]) {
        let Some(_list) = ui.begin_popup(SELECTOR_POPUP) else {
            self.selector_keyboard_started = false;
            self.selector_open = false;
            self.selector_pointer_inside = false;
            return;
        };
        self.selector_open = true;
        self.selector_pointer_inside = ui.is_window_hovered();
        let first_down =
            !self.selector_keyboard_started && ui.is_key_pressed_no_repeat(imgui::Key::DownArrow);
        self.selector_keyboard_started |= first_down
            || ui.is_key_pressed_no_repeat(imgui::Key::UpArrow)
            || ui.is_key_pressed_no_repeat(imgui::Key::Tab);
        for (index, panel) in panels.iter().copied().enumerate() {
            if first_down && index == 0 {
                ui.set_keyboard_focus_here();
            }
            let mut shown = !self.hidden_panels.contains(&(tab, panel));
            let clicked = ui.checkbox(panel.label(), &mut shown);
            let enter = ui.is_item_focused()
                && (ui.is_key_pressed_no_repeat(imgui::Key::Enter)
                    || ui.is_key_pressed_no_repeat(imgui::Key::KeypadEnter));
            if clicked || enter {
                self.selector_keyboard_started = true;
                self.toggle_panel(tab, panel);
            }
        }
    }
}
fn selector_button(ui: &Ui, width: f32) -> bool {
    crate::widget_abi::cursor_position(ui, [width - 30., 35.]);
    let clicked = crate::widget_abi::button(ui, "v##ribbon-panel-selector", [22., 24.]);
    let pressed = ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Left);
    if clicked {
        ui.open_popup(SELECTOR_POPUP);
    }
    if ui.is_item_hovered() {
        ui.tooltip_text("Show Panels");
    }
    clicked || pressed
}
