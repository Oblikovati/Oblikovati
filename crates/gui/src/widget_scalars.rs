// SPDX-License-Identifier: GPL-2.0-only
//! Scope ImGui style and drawing operations without passing vector classes through the ABI.
use imgui::{ImColor32, StyleColor, Ui, sys};

unsafe extern "C" {
    fn oblikovati_gui_push_style_vector(index: i32, x: f32, y: f32);
    fn oblikovati_gui_push_style_color(index: i32, r: f32, g: f32, b: f32, a: f32);
    fn oblikovati_gui_filled_circle(x: f32, y: f32, radius: f32, color: u32);
    fn oblikovati_gui_line(x1: f32, y1: f32, x2: f32, y2: f32, color: u32);
    fn oblikovati_gui_image(texture: usize, width: f32, height: f32);
    fn oblikovati_gui_foreground_dimension(
        x1: f32,
        y1: f32,
        x2: f32,
        y2: f32,
        color: u32,
        label: *const u8,
        length: usize,
    );
}

pub(crate) struct StyleVector<'ui> {
    _ui: &'ui Ui,
}

impl Drop for StyleVector<'_> {
    fn drop(&mut self) {
        // SAFETY: the active frame and its matching style push live for this guard.
        unsafe { sys::igPopStyleVar(1) };
    }
}

pub(crate) fn window_min_size(ui: &Ui, size: [f32; 2]) -> StyleVector<'_> {
    push_vector(ui, sys::ImGuiStyleVar_WindowMinSize, size)
}

pub(crate) fn window_padding(ui: &Ui, padding: [f32; 2]) -> StyleVector<'_> {
    push_vector(ui, sys::ImGuiStyleVar_WindowPadding, padding)
}

pub(crate) fn frame_padding(ui: &Ui, padding: [f32; 2]) -> StyleVector<'_> {
    push_vector(ui, sys::ImGuiStyleVar_FramePadding, padding)
}

fn push_vector(ui: &Ui, style: sys::ImGuiStyleVar_, [x, y]: [f32; 2]) -> StyleVector<'_> {
    // SAFETY: the active frame owns the style stack, and the private callers use vector slots.
    unsafe { oblikovati_gui_push_style_vector(style as i32, x, y) };
    StyleVector { _ui: ui }
}

pub(crate) struct StyleColorGuard<'ui> {
    _ui: &'ui Ui,
}

impl Drop for StyleColorGuard<'_> {
    fn drop(&mut self) {
        // SAFETY: the active frame and its matching color push live for this guard.
        unsafe { sys::igPopStyleColor(1) };
    }
}

pub(crate) fn push_style_color(
    ui: &Ui,
    slot: StyleColor,
    [r, g, b, a]: [f32; 4],
) -> StyleColorGuard<'_> {
    // SAFETY: the active frame owns the color stack, and the caller supplies a valid style slot.
    unsafe { oblikovati_gui_push_style_color(slot as i32, r, g, b, a) };
    StyleColorGuard { _ui: ui }
}

pub(crate) fn filled_circle(_: &Ui, [x, y]: [f32; 2], radius: f32, color: [f32; 4]) {
    // SAFETY: the borrowed UI owns the active window draw list for this call.
    unsafe { oblikovati_gui_filled_circle(x, y, radius, ImColor32::from(color).to_bits()) };
}

pub(crate) fn line(_: &Ui, [x1, y1]: [f32; 2], [x2, y2]: [f32; 2], color: [f32; 4]) {
    // SAFETY: the borrowed UI owns the active window draw list for this call.
    unsafe { oblikovati_gui_line(x1, y1, x2, y2, ImColor32::from(color).to_bits()) };
}

/// Draw a texture with default UVs, tint and border without passing ImGui vectors across the ABI.
pub(crate) fn image(_: &Ui, texture: usize, size: [f32; 2]) {
    // The UI frame owns the draw list and retains the texture ID in its draw commands.
    unsafe { oblikovati_gui_image(texture, size[0], size[1]) };
}

/// Draw the extrusion distance guide, tip and literal label in the foreground layer.
pub(crate) fn foreground_dimension(
    _: &Ui,
    [[x1, y1], [x2, y2]]: [[f32; 2]; 2],
    color: [f32; 4],
    label: &str,
) {
    // The active frame owns the draw list, which copies the borrowed label synchronously.
    unsafe {
        oblikovati_gui_foreground_dimension(
            x1,
            y1,
            x2,
            y2,
            ImColor32::from(color).to_bits(),
            label.as_ptr(),
            label.len(),
        )
    };
}
