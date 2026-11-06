// SPDX-License-Identifier: GPL-2.0-only
use imgui::{Condition, ImColor32, ImString, SelectableFlags, Ui};
unsafe extern "C" {
    fn oblikovati_gui_fit_dialog(title: *const u8, x: f32, y: f32, width: f32, height: f32);
    fn oblikovati_gui_begin_scroll_region(identity: *const u8, width: f32, height: f32) -> bool;
    fn oblikovati_gui_dummy(width: f32, height: f32);
    fn oblikovati_gui_window_position(x: f32, y: f32, condition: i32);
    fn oblikovati_gui_window_size(width: f32, height: f32, condition: i32);
    fn oblikovati_gui_cursor_position(x: f32, y: f32);
    fn oblikovati_gui_button(label: *const u8, width: f32, height: f32, selected: bool) -> bool;
    fn oblikovati_gui_selectable(label: *const u8, selected: bool, flags: i32) -> bool;
    fn oblikovati_gui_literal_text(
        text: *const u8,
        length: usize,
        x: f32,
        y: f32,
        color: u32,
        background: bool,
    );
}
/// Prepare a window through scalar ABI calls; example: `window_geometry(ui, [0.,0.], [640.,480.], Condition::Always)`.
pub(crate) fn window_geometry(_: &Ui, position: [f32; 2], size: [f32; 2], condition: Condition) {
    if condition == Condition::Never {
        return;
    }
    // The live Ui owns this thread's context; only scalar arguments cross into C++ (#3803).
    unsafe {
        oblikovati_gui_window_position(position[0], position[1], condition as i32);
        oblikovati_gui_window_size(size[0], size[1], condition as i32);
    }
}
/// Move the current window cursor; example: `cursor_position(ui, [16.,59.])` before a ribbon control.
pub(crate) fn cursor_position(_: &Ui, position: [f32; 2]) {
    // The Ui borrow keeps the current context and window alive for this scalar call.
    unsafe { oblikovati_gui_cursor_position(position[0], position[1]) };
}
/// Draw a sized command control; example: `button(ui, "Save", [0.;2])` uses natural sizing.
pub(crate) fn button(ui: &Ui, label: &str, size: [f32; 2]) -> bool {
    toggle_button(ui, label, size, false)
}
/// Draw an owned mode's pressed state through scalars; example: `toggle_button(ui, "Construction", [108.,18.], true)`.
pub(crate) fn toggle_button(_: &Ui, label: &str, size: [f32; 2], selected: bool) -> bool {
    let label = ImString::new(label);
    // ImString owns terminated UTF-8 bytes through the call; C++ retains no label pointer.
    unsafe { oblikovati_gui_button(label.as_ptr().cast(), size[0], size[1], selected) }
}
/// Draw a current-window row; example: `selectable(ui, "##row", true, true)` permits double-click activation.
pub(crate) fn selectable(_: &Ui, label: &str, selected: bool, double_click: bool) -> bool {
    let flags = if double_click {
        SelectableFlags::ALLOW_DOUBLE_CLICK.bits()
    } else {
        0
    };
    let label = ImString::new(label);
    // The label owner and Ui remain live; no C++ vector class crosses the ABI.
    unsafe { oblikovati_gui_selectable(label.as_ptr().cast(), selected, flags as i32) }
}
/// Draw literal UTF-8 in the current window; example: `literal_text(ui, point, color, "Part##1")` preserves hashes.
pub(crate) fn literal_text(ui: &Ui, position: [f32; 2], color: [f32; 4], text: &str) {
    draw_literal_text(ui, position, color, text, false);
}
/// Draw below windows without a vector ABI; example: `background_literal_text(ui, point, color, "d0 = 10 mm")`.
pub(crate) fn background_literal_text(ui: &Ui, position: [f32; 2], color: [f32; 4], text: &str) {
    draw_literal_text(ui, position, color, text, true);
}
fn draw_literal_text(_: &Ui, position: [f32; 2], color: [f32; 4], text: &str, background: bool) {
    let color: ImColor32 = color.into();
    // The byte slice outlives AddText, which copies its glyph vertices into the current draw list.
    unsafe {
        oblikovati_gui_literal_text(
            text.as_ptr(),
            text.len(),
            position[0],
            position[1],
            color.to_bits(),
            background,
        )
    };
}

#[cfg(test)]
#[path = "../tests/widget_abi/native.rs"]
mod tests;

/// A live child window is always ended, including clipped frames and unwinding.
pub(crate) struct ScrollingRegion<'ui> {
    pub visible: bool,
    _ui: &'ui Ui,
}
impl Drop for ScrollingRegion<'_> {
    fn drop(&mut self) {
        // SAFETY: this token retains the UI that opened exactly one child, including a false BeginChild result.
        unsafe { imgui::sys::igEndChild() };
    }
}
/// Open scalar-sized scrolling content; example: `let region = scroll_region(ui, "panels", [640.,80.]);`.
pub(crate) fn scroll_region<'ui>(
    ui: &'ui Ui,
    identity: &str,
    size: [f32; 2],
) -> ScrollingRegion<'ui> {
    let identity = ImString::new(identity);
    // SAFETY: the UI owns the current context; C++ copies the live ID and constructs its vector (#3803, #3849).
    let visible =
        unsafe { oblikovati_gui_begin_scroll_region(identity.as_ptr().cast(), size[0], size[1]) };
    ScrollingRegion { visible, _ui: ui }
}
/// Reserve layout space through scalar ABI; example: `dummy(ui, [364.,0.])` keeps the Create grid's width.
pub(crate) fn dummy(_: &Ui, size: [f32; 2]) {
    // SAFETY: the owning frame retains the context; only C++ constructs the aggregate argument (#3803).
    unsafe { oblikovati_gui_dummy(size[0], size[1]) };
}

/// Keep a floating dialog inside the current content viewport while preserving valid user placement.
/// Example: `fit_dialog(ui, "Parameters", [320.,8.], [540.,360.])` before opening that window.
pub(crate) fn fit_dialog(_: &Ui, title: &str, origin: [f32; 2], size: [f32; 2]) {
    let title = ImString::new(title);
    // SAFETY: the frame owns the current context; the live ID and scalar values are copied by C++ (#3803, #3850).
    unsafe {
        oblikovati_gui_fit_dialog(
            title.as_ptr().cast(),
            origin[0],
            origin[1],
            size[0],
            size[1],
        )
    };
}
