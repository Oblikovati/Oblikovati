// SPDX-License-Identifier: GPL-2.0-only
//! The primary command and dropdown share one allotted ribbon cell.
use crate::icons::GuiRibbonIcons;
use imgui::Ui;

const TRIGGER_WIDTH: f32 = 16.;
const TRIGGER_GAP: f32 = 2.;
pub(crate) const TRIGGER_SPACE: f32 = TRIGGER_WIDTH + TRIGGER_GAP;

pub(crate) fn primary_size([width, height]: [f32; 2]) -> [f32; 2] {
    [width - TRIGGER_SPACE, height]
}

pub(crate) fn trigger(ui: &Ui, icons: &GuiRibbonIcons, id: &str, height: f32) -> bool {
    ui.same_line_with_spacing(0., TRIGGER_GAP * icons.layout_factor());
    crate::widget_abi::button(ui, &format!("▼{id}"), icons.cell([TRIGGER_WIDTH, height]))
}
