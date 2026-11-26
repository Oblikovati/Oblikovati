// SPDX-License-Identifier: GPL-2.0-only
use super::{LINE_COLOR, drawing_lines::screen_point};
use crate::GuiDrawingProjection;
use oblikovati_model::drawing::DrawingBaseViewDefinition;

pub(super) fn draw(
    ui: &imgui::Ui,
    list: &imgui::DrawListMut<'_>,
    views: &[DrawingBaseViewDefinition],
    projection: Option<&GuiDrawingProjection>,
    [minimum, maximum]: [[f32; 2]; 2],
    dimensions: [f64; 2],
) {
    list.with_clip_rect_intersect(minimum, maximum, || {
        for view in views {
            let label = view.label();
            if label.is_empty() {
                continue;
            }
            let [width, height] = ui.calc_text_size(&label);
            let position = view.label_position_mm();
            let origin = if position == [0., 0.] {
                let bottom = projection
                    .and_then(|projection| projection.view_bottoms_mm.get(&view.name))
                    .copied()
                    .unwrap_or(view.center_y_mm);
                let [x, y] = screen_point([view.center_x_mm, bottom], minimum, maximum, dimensions);
                [x - width * 0.5, y + 4.]
            } else {
                let [x, y] = screen_point(position, minimum, maximum, dimensions);
                [x - width * 0.5, y - height * 0.5]
            };
            if origin.iter().all(|coordinate| coordinate.is_finite()) {
                list.add_text(origin, LINE_COLOR, &label);
            }
        }
    });
}
