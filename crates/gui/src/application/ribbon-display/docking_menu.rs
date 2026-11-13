// SPDX-License-Identifier: GPL-2.0-only
//! Docking choices stay closed after Escape until the pointer leaves their parent row.
use super::{DOCKING_POSITIONS, RibbonDisplayState};
use imgui::{Key, StyleColor, Ui};

const DOCKING_MENU: &str = "##ribbon-docking-positions";

impl RibbonDisplayState {
    pub(super) fn docking_menu(&mut self, ui: &Ui) {
        if self.docking_open && ui.is_key_pressed_no_repeat(Key::Escape) {
            self.docking_hover_blocked = true;
        }
        let (position, open) = self.docking_trigger(ui);
        if open {
            self.docking_hover_blocked = false;
            ui.open_popup(DOCKING_MENU);
        }
        crate::widget_abi::next_window_position(ui, position);
        self.docking_open = self.draw_docking_choices(ui);
    }

    fn docking_trigger(&mut self, ui: &Ui) -> ([f32; 2], bool) {
        let pressed = crate::widget_abi::dialog_selectable(ui, "Docking Positions", false);
        let hovered = crate::widget_abi::item_mouse_hovered(ui);
        let keyboard = ui.is_item_focused() && ui.is_key_pressed_no_repeat(Key::RightArrow);
        let top = ui.item_rect_min();
        let bottom = ui.item_rect_max();
        let arrow = [bottom[0] - ui.current_font_size(), top[1]];
        crate::widget_abi::literal_text(ui, arrow, ui.style_color(StyleColor::Text), ">");
        if !hovered {
            self.docking_hover_blocked = false;
        }
        let open =
            pressed || keyboard || (hovered && !self.docking_open && !self.docking_hover_blocked);
        ([bottom[0], top[1]], open)
    }

    fn draw_docking_choices(&mut self, ui: &Ui) -> bool {
        let Some(_menu) = ui.begin_popup(DOCKING_MENU) else {
            return false;
        };
        self.capture_submenu_bounds(ui);
        if ui.is_key_pressed_no_repeat(Key::LeftArrow) {
            self.docking_hover_blocked = true;
            ui.close_current_popup();
            return false;
        }
        for (label, placement) in DOCKING_POSITIONS {
            if ui
                .menu_item_config(label)
                .selected(self.placement == placement)
                .build()
            {
                self.place(placement);
                self.close_context_menu = true;
            }
        }
        true
    }
}
