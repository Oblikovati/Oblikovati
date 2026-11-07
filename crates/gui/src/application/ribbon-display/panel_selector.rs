// SPDX-License-Identifier: GPL-2.0-only
//! The direct panel list shares visibility with the context menu but stays open on a change (#3856).
use super::*;
pub(crate) const PANEL_SELECTOR_WIDTH: f32 = 32.;
const SELECTOR_POPUP: &str = "##ribbon-panel-selector-list";
impl RibbonDisplayState {
    /// Toggle panels without reopening the list; example: `display.panel_selector(ui, tab, state)`.
    pub(crate) fn panel_selector(
        &mut self,
        ui: &Ui,
        tab: RibbonTab,
        state: &crate::GuiCommandState,
    ) {
        if self.mode != RibbonDisplay::Full || self.placement != RibbonPlacement::Top {
            return;
        }
        let width = ui.window_size()[0];
        selector_button(ui, width);
        let origin = ui.window_pos();
        crate::widget_abi::next_window_position(ui, [origin[0] + width - 180., origin[1] + 60.]);
        self.selector_choices(ui, tab, state);
    }
    fn selector_choices(&mut self, ui: &Ui, tab: RibbonTab, state: &crate::GuiCommandState) {
        let Some(_list) = ui.begin_popup(SELECTOR_POPUP) else {
            return;
        };
        for panel in crate::ribbon_panels::panels_for(tab, state) {
            let mut shown = !self.hidden_panels.contains(&(tab, panel));
            if ui.checkbox(panel.label(), &mut shown) {
                self.toggle_panel(tab, panel);
            }
        }
    }
}
fn selector_button(ui: &Ui, width: f32) {
    crate::widget_abi::cursor_position(ui, [width - 30., 35.]);
    if crate::widget_abi::button(ui, "v##ribbon-panel-selector", [22., 24.]) {
        ui.open_popup(SELECTOR_POPUP);
    }
    if ui.is_item_hovered() {
        ui.tooltip_text("Show Panels");
    }
}
