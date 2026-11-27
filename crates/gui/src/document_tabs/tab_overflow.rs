// SPDX-License-Identifier: GPL-2.0-only
//! Fit and scroll the document tab strip when it overflows.
use imgui::{ColorStackToken, StyleColor, Ui};

/// Room for the two scroll arrows, taken from the strip only when it overflows.
pub(super) const ARROWS_WIDTH: f32 = 52.;

/// Keep ImGui's automatic visible selection neutral; example: `neutral_selection(ui)`.
pub(super) fn neutral_selection(ui: &Ui) -> [ColorStackToken<'_>; 2] {
    [
        ui.push_style_color(StyleColor::TabActive, ui.style_color(StyleColor::Tab)),
        ui.push_style_color(
            StyleColor::TabUnfocusedActive,
            ui.style_color(StyleColor::TabUnfocused),
        ),
    ]
}

/// Which tabs fit from `first`; a tab just brought forward is scrolled into the strip.
pub(super) fn visible_tabs(
    widths: &[f32],
    room: f32,
    first: usize,
    active: Option<usize>,
    reveal: bool,
) -> (usize, usize) {
    let mut first = first.min(widths.len().saturating_sub(1));
    if let Some(active) = active.filter(|_| reveal) {
        first = first.min(active);
        while first < active && fitting(widths, first, room) + first <= active {
            first += 1;
        }
    }
    (first, fitting(widths, first, room))
}
fn fitting(widths: &[f32], first: usize, room: f32) -> usize {
    let mut used = 0.;
    let mut count = 0;
    for width in &widths[first..] {
        used += width;
        if used > room && count > 0 {
            break;
        }
        count += 1;
    }
    count
}
/// Scrolling only moves the strip: the active document stays what it is.
pub(super) fn scroll_arrows(
    ui: &Ui,
    available: f32,
    first: &mut usize,
    moved: &mut bool,
    more_to_the_right: bool,
) {
    let start = ui.window_content_region_min()[0];
    ui.set_cursor_pos([
        start + available - ARROWS_WIDTH + 4.,
        ui.window_content_region_min()[1],
    ]);
    {
        let _at_start = ui.begin_disabled(*first == 0);
        if ui.button("<##tabs-left") {
            *first = first.saturating_sub(1);
            *moved = true;
        }
    }
    ui.same_line();
    let _at_end = ui.begin_disabled(!more_to_the_right);
    if ui.button(">##tabs-right") {
        *first += 1;
        *moved = true;
    }
}
#[cfg(test)]
mod tests {
    use super::visible_tabs;

    #[test]
    fn active_tab_is_revealed_only_when_requested() {
        let widths = [40., 40., 40.];
        assert_eq!(visible_tabs(&widths, 80., 0, Some(2), false), (0, 2));
        assert_eq!(visible_tabs(&widths, 80., 0, Some(2), true), (1, 2));
        assert_eq!(visible_tabs(&widths, 10., 9, None, false), (2, 1));
    }
}
