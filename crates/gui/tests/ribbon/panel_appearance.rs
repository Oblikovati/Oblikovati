// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::ribbon_command_layout::RibbonAppearance;

#[test]
fn appearance_change_discards_previous_panel_measurements() {
    let mut layout = RibbonPanelLayout::default();
    layout.use_appearance(RibbonAppearance::Normal);
    layout.record_width(RibbonTab::Model, RibbonPanel::PartModify, 200.);
    layout.pending_layout = false;
    layout.use_appearance(RibbonAppearance::Normal);
    assert_eq!(layout.widths.len(), 1);
    assert!(!layout.pending_layout);
    layout.use_appearance(RibbonAppearance::TextOff);
    assert!(layout.widths.is_empty());
    assert!(layout.pending_layout);
}
