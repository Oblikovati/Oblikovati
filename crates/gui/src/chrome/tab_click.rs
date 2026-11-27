// SPDX-License-Identifier: GPL-2.0-only
//! Selected-tab presses while temporary commands are open (#3856).
use super::*;
use crate::ribbon::RibbonTab;

/// Dear ImGui selects tabs on either edge of a right click. Keep the active ribbon
/// selected while its context menu opens, including the release over that menu.
pub(super) fn context_button_event(ui: &Ui) -> bool {
    use imgui::{MouseButton, WindowHoveredFlags};
    (ui.is_mouse_clicked(MouseButton::Right) || ui.is_mouse_released(MouseButton::Right))
        && ui.is_window_hovered_with_flags(
            WindowHoveredFlags::CHILD_WINDOWS | WindowHoveredFlags::ALLOW_WHEN_BLOCKED_BY_POPUP,
        )
}

pub(super) fn temporary_expansion_press(
    ribbon: &mut DocumentRibbon,
    tab: RibbonTab,
    clicked: bool,
    double_click: bool,
) {
    if !clicked
        || double_click
        || ribbon.active != tab
        || ribbon.display.mode != RibbonDisplay::Tabs
        || ribbon.display.temporary != Some(TemporaryPanel::WholeTab)
    {
        return;
    }
    match ribbon.panels.temporary_tab_press() {
        Some(true) => ribbon.display.close_temporary_from_tab(),
        Some(false) => ribbon.display.keep_temporary_on_click = true,
        None => {}
    }
}
