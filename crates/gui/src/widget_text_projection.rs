// SPDX-License-Identifier: GPL-2.0-only
use imgui::Ui;

/// Map a centered label onto the given screen-space basis vectors.
pub(crate) fn projected_text(
    ui: &Ui,
    center: [f32; 2],
    color: [f32; 4],
    text: &str,
    [right, down]: [[f32; 2]; 2],
) {
    let [width, height] = ui.calc_text_size(text);
    // Build the unprojected glyphs away from the window edge to keep them from clipping early.
    let position = ui.window_pos();
    let size = ui.window_size();
    let source: [f32; 2] = std::array::from_fn(|axis| position[axis] + size[axis] / 2.);
    let list = ui.get_window_draw_list();
    // SAFETY: the current window's list is live for this frame, and these reads and writes stay
    // inside the vertices this call appended; nothing else draws between the two reads.
    unsafe {
        let raw = imgui::sys::igGetWindowDrawList();
        let first = (*raw).VtxBuffer.Size;
        list.add_text(
            [source[0] - width / 2., source[1] - height / 2.],
            color,
            text,
        );
        let last = (*raw).VtxBuffer.Size;
        for index in first..last {
            let vertex = &mut *(*raw).VtxBuffer.Data.add(index as usize);
            let (x, y) = (vertex.pos.x - source[0], vertex.pos.y - source[1]);
            vertex.pos.x = center[0] + x * right[0] + y * down[0];
            vertex.pos.y = center[1] + x * right[1] + y * down[1];
        }
    }
}
