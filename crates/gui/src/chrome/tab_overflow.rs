// SPDX-License-Identifier: GPL-2.0-only
//! Fit ribbon tabs before the display controls and offer the rest in a menu.
use super::*;
use crate::ribbon::RibbonTab;

const MINIMUM_OVERFLOW_WIDTH: f32 = 28.;

/// Draw the fitting tabs and the remaining choices; for example, View at a narrow width (#3856).
pub(super) fn draw(ui: &Ui, ribbon: &mut DocumentRibbon, frame: &RibbonFrame) {
    let style = ui.clone_style();
    let spacing = style.item_spacing[0];
    let control_width = 160. + crate::application::ribbon_display::PANEL_SELECTOR_WIDTH;
    let overflow_width =
        (ui.calc_text_size(">>")[0] + 2. * style.frame_padding[0]).max(MINIMUM_OVERFLOW_WIDTH);
    let room = (ui.window_size()[0] - control_width - ui.cursor_pos()[0]).max(1.);
    let widths = tab_widths(ui, &frame.tabs);
    let count = fitting_tabs(&widths, room, overflow_width + spacing);
    let hidden = &frame.tabs[count..];
    let tabs_width = if hidden.is_empty() {
        room
    } else {
        (room - overflow_width - spacing).max(1.)
    };
    let visible = &frame.tabs[..count];
    let changed = ribbon.tabs_changed(visible) || ribbon.tabs_settling();
    let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([0., 0.]));
    ui.child_window("##ribbon-tab-region")
        .size([tabs_width, tab_row_height(ui)])
        .scroll_bar(false)
        .scrollable(false)
        .build(|| draw_visible(ui, ribbon, frame, visible, changed));
    if !hidden.is_empty() {
        ui.same_line();
        overflow_menu(ui, ribbon, hidden, overflow_width);
    }
}

fn tab_widths(ui: &Ui, tabs: &[RibbonTab]) -> Vec<f32> {
    let style = ui.clone_style();
    tabs.iter()
        .map(|tab| {
            ui.calc_text_size(tab.label())[0] + 2. * style.frame_padding[0] + style.item_spacing[0]
        })
        .collect()
}

fn fitting_tabs(widths: &[f32], room: f32, reserve: f32) -> usize {
    if widths.iter().sum::<f32>() <= room {
        return widths.len();
    }
    let mut used = 0.;
    widths
        .iter()
        .take_while(|width| {
            used += **width;
            used <= (room - reserve).max(0.)
        })
        .count()
}

fn draw_visible(
    ui: &Ui,
    ribbon: &mut DocumentRibbon,
    frame: &RibbonFrame,
    visible: &[RibbonTab],
    changed: bool,
) {
    let _neutral = (!visible.contains(&ribbon.active)).then(|| {
        [
            ui.push_style_color(
                imgui::StyleColor::TabActive,
                ui.style_color(imgui::StyleColor::Tab),
            ),
            ui.push_style_color(
                imgui::StyleColor::TabUnfocusedActive,
                ui.style_color(imgui::StyleColor::TabUnfocused),
            ),
        ]
    });
    super::ribbon_tabs(ui, ribbon, frame, visible, changed);
}

fn overflow_menu(ui: &Ui, ribbon: &mut DocumentRibbon, hidden: &[RibbonTab], width: f32) {
    if crate::widget_abi::button(ui, ">>##ribbon-tab-overflow", [width, 0.]) {
        ui.open_popup("##ribbon-tab-overflow-menu");
    }
    if let Some(_menu) = ui.begin_popup("##ribbon-tab-overflow-menu") {
        for tab in hidden {
            if ui.menu_item(tab.label()) {
                ribbon.select_tab(*tab);
            }
        }
    }
}

pub(super) fn tab_row_height(ui: &Ui) -> f32 {
    ui.frame_height().max(20.) + 4.
}

#[cfg(test)]
mod tests {
    use super::fitting_tabs;

    #[test]
    fn reserves_menu_only_when_tabs_do_not_fit() {
        assert_eq!(fitting_tabs(&[40., 50., 60.], 150., 20.), 3);
        assert_eq!(fitting_tabs(&[40., 50., 60.], 149., 20.), 2);
        assert_eq!(fitting_tabs(&[40., 50., 60.], 59., 20.), 0);
        assert_eq!(fitting_tabs(&[], 0., 20.), 0);
    }
}
