// SPDX-License-Identifier: GPL-2.0-only
//! Fit ribbon tabs before the display controls and offer the rest in a menu.
use crate::application::ribbon_display::{DENSITY_CONTROL_WIDTH, PANEL_SELECTOR_WIDTH};
pub(super) const RIBBON_CONTROL_WIDTH: f32 = DENSITY_CONTROL_WIDTH + PANEL_SELECTOR_WIDTH;
use super::*;
use crate::ribbon::RibbonTab;
use imgui::TabItemFlags;

const MINIMUM_OVERFLOW_WIDTH: f32 = 28.;

/// Draw the fitting tabs and the remaining choices; for example, View at a narrow width (#3856).
pub(super) fn draw(ui: &Ui, ribbon: &mut DocumentRibbon, frame: &RibbonFrame) {
    let style = ui.clone_style();
    let spacing = style.item_spacing[0];
    let overflow_width =
        (ui.calc_text_size(">>")[0] + 2. * style.frame_padding[0]).max(MINIMUM_OVERFLOW_WIDTH);
    let room = (ui.window_size()[0] - RIBBON_CONTROL_WIDTH - ui.cursor_pos()[0]).max(1.);
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
    ribbon_tabs(ui, ribbon, frame, visible, changed);
}

fn ribbon_tabs(
    ui: &Ui,
    ribbon: &mut DocumentRibbon,
    frame: &RibbonFrame,
    visible: &[crate::ribbon::RibbonTab],
    changed: bool,
) {
    let Some(_tabs) = ui.tab_bar("##document-ribbon-tabs") else {
        return;
    };
    let mut hovered = false;
    let mut selector_tab = None;
    let hidden_active = !visible.contains(&ribbon.active);
    let context_click = tab_click::context_button_event(ui);
    let restore = (changed || context_click).then_some(ribbon.active);
    let requested = frame.select.or(restore);
    let mut bounds = Vec::with_capacity(visible.len());
    for tab in visible {
        let selected = requested == Some(*tab);
        let flags = if selected {
            TabItemFlags::SET_SELECTED
        } else {
            TabItemFlags::empty()
        };
        let item = ui.tab_item_with_flags(tab.label(), None, flags);
        bounds.push((*tab, ui.item_rect_min(), ui.item_rect_max()));
        if ribbon.display.selector_pointer_press(ui) || ribbon.display.context_tab_press(ui) {
            selector_tab = Some(*tab);
        }
        let tip = crate::ribbon_key_tips::tab_tip(*tab).to_string();
        if ribbon.tips.tabs() && ribbon.tips.matching(&tip) {
            crate::ribbon_key_tips::badge(ui, &tip);
        }
        hovered |= ui.is_item_hovered();
        // On release: the press itself is a click outside any panel still open, and closes it.
        let released = ui.is_item_hovered() && ui.is_mouse_released(imgui::MouseButton::Left);
        let clicked = ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Left);
        let through_popup = ribbon.tab_popup_press.released(ui, *tab);
        let double_click = clicked && ui.is_mouse_double_clicked(imgui::MouseButton::Left);
        tab_click::temporary_expansion_press(ribbon, *tab, clicked, double_click);
        if clicked && ribbon.active != *tab {
            ribbon.display.switch_from_open_compact_panel();
        }
        let doubled = double_click && ribbon.double_click_tab == Some(*tab);
        if clicked && !double_click {
            ribbon.double_click_tab = (ribbon.active == *tab).then_some(*tab);
        }
        if double_click {
            ribbon.double_click_tab = None;
        }
        if (hidden_active && clicked) || through_popup {
            ribbon.select_tab(*tab);
        }
        if item.is_some()
            && (!hidden_active || clicked || selected)
            && (requested.is_none() || clicked || selected)
        {
            ribbon.active = *tab;
        }
        drop(item);
        if doubled {
            ribbon.display.advance_from_tab();
        }
        if released && ribbon.display.take_tab_release() {
            ribbon.display.open_temporary(TemporaryPanel::WholeTab);
        }
    }
    ribbon.tab_popup_press.finish_frame(ui);
    if let Some(tab) = selector_tab.or(context_click.then_some(ribbon.active)) {
        ribbon.select_tab(tab);
    }
    if hovered {
        ribbon.scroll_tabs(&frame.tabs, ui.io().mouse_wheel);
    }
    if let Some((_, minimum, maximum)) = bounds.iter().find(|(tab, _, _)| *tab == ribbon.active) {
        crate::theme::tab_underline_at(ui, *minimum, *maximum);
    }
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
