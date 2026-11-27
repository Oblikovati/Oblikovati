// SPDX-License-Identifier: GPL-2.0-only
//! Keep ImGui next-window positions, pivots and sizes scalar across the Rust boundary.
use imgui::{Condition, Ui};

unsafe extern "C" {
    fn oblikovati_gui_window_position(x: f32, y: f32, pivot_x: f32, pivot_y: f32, condition: i32);
    fn oblikovati_gui_window_size(width: f32, height: f32, condition: i32);
}

/// Prepare a window through scalar ABI calls; example: `window_geometry(ui, [0.,0.], [640.,480.], Condition::Always)`.
pub(crate) fn window_geometry(ui: &Ui, position: [f32; 2], size: [f32; 2], condition: Condition) {
    window_position(ui, position, condition);
    window_size(ui, size, condition);
}
/// Set just the origin, for example `window_position(ui, [24., 42.], Condition::Always)`.
pub(crate) fn window_position(ui: &Ui, position: [f32; 2], condition: Condition) {
    window_position_with_pivot(ui, position, [0.; 2], condition);
}
/// Anchor a window by a corner or edge, for example `window_position_with_pivot(ui, at, [0.5, 1.], Condition::Always)`.
pub(crate) fn window_position_with_pivot(
    _: &Ui,
    [x, y]: [f32; 2],
    [pivot_x, pivot_y]: [f32; 2],
    condition: Condition,
) {
    if condition != Condition::Never {
        // The live Ui owns this thread's context; only scalars cross into C++ (#3803).
        unsafe { oblikovati_gui_window_position(x, y, pivot_x, pivot_y, condition as i32) };
    }
}
/// Set just the size, for example `window_size(ui, [640., 480.], Condition::FirstUseEver)`.
pub(crate) fn window_size(_: &Ui, [width, height]: [f32; 2], condition: Condition) {
    if condition != Condition::Never {
        // The live Ui owns this thread's context; only scalars cross into C++ (#3803).
        unsafe { oblikovati_gui_window_size(width, height, condition as i32) };
    }
}
/// Place the next window where it first appears; example: a list opened beside its menu row.
pub(crate) fn next_window_position(ui: &Ui, position: [f32; 2]) {
    window_position(ui, position, Condition::Appearing);
}
