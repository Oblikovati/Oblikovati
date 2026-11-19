// SPDX-License-Identifier: GPL-2.0-only
//! Draw the active sketch's projected coordinate axes behind its controls.
use crate::GuiFrameRequest;
use imgui::Ui;

// Match the global orientation triad so an axis keeps its identity (#4148).
const COLORS: [[f32; 4]; 2] = [[0.92, 0.18, 0.18, 1.], [0.20, 0.80, 0.28, 1.]];

pub(crate) fn draw(ui: &Ui, request: &GuiFrameRequest) {
    let scale = request.layout_scale();
    let logical = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let list = ui.get_background_draw_list();
    for (line, color) in request.commands.sketch_axes.iter().zip(COLORS) {
        list.add_line(logical(line[0]), logical(line[1]), color)
            .build();
    }
}
