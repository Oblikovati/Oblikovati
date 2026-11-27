// SPDX-License-Identifier: GPL-2.0-only
//! The primary command and dropdown share one allotted ribbon cell.
use crate::icons::GuiRibbonIcons;
use imgui::Ui;

const TRIGGER_WIDTH: f32 = 14.;
const TRIGGER_GAP: f32 = 0.;
pub(crate) const TRIGGER_SPACE: f32 = TRIGGER_WIDTH + TRIGGER_GAP;

pub(crate) fn primary_size([width, height]: [f32; 2]) -> [f32; 2] {
    [width - TRIGGER_SPACE, height]
}

pub(crate) fn trigger(ui: &Ui, icons: &GuiRibbonIcons, id: &str, height: f32) -> bool {
    ui.same_line_with_spacing(0., TRIGGER_GAP * icons.layout_factor());
    let _bg = ui.push_style_color(imgui::StyleColor::Button, [0.; 4]);
    let _border = ui.push_style_var(imgui::StyleVar::FrameBorderSize(0.));
    let _ink = ui.push_style_color(imgui::StyleColor::Text, crate::theme::text_muted(ui));
    let clicked =
        crate::widget_abi::button(ui, &format!("▼{id}"), icons.cell([TRIGGER_WIDTH, height]));
    if ui.is_item_hovered() || ui.is_item_active() {
        let [lo, hi] = [ui.item_rect_min(), ui.item_rect_max()];
        let draw = ui.get_window_draw_list();
        draw.add_line(
            [lo[0], lo[1] + 4.],
            [lo[0], hi[1] - 4.],
            crate::theme::border_subtle(ui),
        )
        .build();
        draw.add_rect(lo, hi, ui.style_color(imgui::StyleColor::Border))
            .rounding(ui.clone_style().frame_rounding)
            .build();
    }
    clicked
}
