// SPDX-License-Identifier: GPL-2.0-only
//! Selected-tab presses while temporary commands are open (#3856).
use super::*;
use crate::ribbon::RibbonTab;

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
