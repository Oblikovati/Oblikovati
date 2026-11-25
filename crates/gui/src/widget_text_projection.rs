// SPDX-License-Identifier: GPL-2.0-only
use imgui::{DrawListMut, Ui, sys};

/// Map a centered label onto the given screen-space basis vectors.
pub(crate) fn projected_text(
    ui: &Ui,
    center: [f32; 2],
    color: [f32; 4],
    text: &str,
    [right, down]: [[f32; 2]; 2],
) {
    let position = ui.window_pos();
    let size = ui.window_size();
    let source = std::array::from_fn(|axis| position[axis] + size[axis] / 2.);
    let list = ui.get_window_draw_list();
    // The live Ui backs both handles for the same draw list.
    let raw = unsafe { sys::igGetWindowDrawList() };
    transform_label(ui, &list, raw, source, color, text, |[x, y]| {
        std::array::from_fn(|axis| center[axis] + x * right[axis] + y * down[axis])
    });
}

/// Project a label on the background list, using a source point inside its current clip.
pub(crate) fn projected_background_text(
    ui: &Ui,
    draw: &DrawListMut<'_>,
    source: [f32; 2],
    color: [f32; 4],
    text: &str,
    project: impl Fn([f32; 2]) -> [f32; 2],
) {
    // Callers pass the current frame's background list, which owns the clipping rectangle.
    let raw = unsafe { sys::igGetBackgroundDrawList_Nil() };
    transform_label(ui, draw, raw, source, color, text, project);
}

fn transform_label(
    ui: &Ui,
    draw: &DrawListMut<'_>,
    raw: *mut sys::ImDrawList,
    source: [f32; 2],
    color: [f32; 4],
    text: &str,
    project: impl Fn([f32; 2]) -> [f32; 2],
) {
    let [width, height] = ui.calc_text_size(text);
    // SAFETY: the paired handles name a live draw list. Only this call's appended vertices change.
    unsafe {
        let first = (*raw).VtxBuffer.Size;
        draw.add_text(
            [source[0] - width / 2., source[1] - height / 2.],
            color,
            text,
        );
        for index in first..(*raw).VtxBuffer.Size {
            let vertex = &mut *(*raw).VtxBuffer.Data.add(index as usize);
            let point = project([vertex.pos.x - source[0], vertex.pos.y - source[1]]);
            vertex.pos.x = point[0];
            vertex.pos.y = point[1];
        }
    }
}
