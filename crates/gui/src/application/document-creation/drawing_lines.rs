// SPDX-License-Identifier: GPL-2.0-only
//! Draw sheet-space samples without giving display geometry back to the model.
use crate::GuiDrawingProjection;

pub(super) fn draw(
    ui: &imgui::Ui,
    list: &imgui::DrawListMut<'_>,
    projection: &GuiDrawingProjection,
    minimum: [f32; 2],
    maximum: [f32; 2],
    dimensions: [f64; 2],
) {
    list.with_clip_rect_intersect(minimum, maximum, || {
        for line in &projection.lines {
            for segment in line.points.windows(2) {
                let Some([a, b]) = clip_segment(segment[0], segment[1], dimensions) else {
                    continue;
                };
                let a = screen_point(a, minimum, maximum, dimensions);
                let b = screen_point(b, minimum, maximum, dimensions);
                let color = if line.preview {
                    [0.1, 0.35, 0.85, 1.]
                } else {
                    [0.1, 0.1, 0.12, 1.]
                };
                if line.hidden {
                    dashed(list, a, b, color);
                } else {
                    list.add_line(a, b, color).build();
                }
            }
        }
        for table in &projection.parts_lists {
            draw_parts_list(ui, list, table, minimum, maximum, dimensions);
        }
        if let Some(error) = &projection.error {
            list.add_text(
                [minimum[0] + 8., minimum[1] + 8.],
                [0.7, 0.1, 0.1, 1.],
                error,
            );
        }
    });
}
fn draw_parts_list(
    ui: &imgui::Ui,
    list: &imgui::DrawListMut<'_>,
    table: &oblikovati_model::drawing::DrawingPartsListPresentation,
    minimum: [f32; 2],
    maximum: [f32; 2],
    dimensions: [f64; 2],
) {
    const INK: [f32; 4] = [0.1, 0.1, 0.12, 1.];
    let to_screen = |point| screen_point(point, minimum, maximum, dimensions);
    for line in &table.lines {
        if let Some([start, end]) = clip_segment(line.start, line.end, dimensions) {
            list.add_line(to_screen(start), to_screen(end), INK).build();
        }
    }
    for label in &table.labels {
        let [x, y] = label.center;
        let [w, h] = label.cell_size.map(|length| length / 2.);
        let left_top = to_screen([x - w, y + h]);
        let right_bottom = to_screen([x + w, y - h]);
        let [width, height] = ui.calc_text_size(&label.text);
        let [cx, cy] = to_screen(label.center);
        list.with_clip_rect_intersect(left_top, right_bottom, || {
            let origin = [(cx - width * 0.5).max(left_top[0] + 2.), cy - height * 0.5];
            list.add_text(origin, INK, &label.text);
        });
    }
}
fn screen_point(
    point: [f64; 2],
    minimum: [f32; 2],
    maximum: [f32; 2],
    dimensions: [f64; 2],
) -> [f32; 2] {
    [
        minimum[0] + (point[0] / dimensions[0]) as f32 * (maximum[0] - minimum[0]),
        maximum[1] - (point[1] / dimensions[1]) as f32 * (maximum[1] - minimum[1]),
    ]
}
fn clip_segment(a: [f64; 2], b: [f64; 2], dimensions: [f64; 2]) -> Option<[[f64; 2]; 2]> {
    if a.iter()
        .chain(&b)
        .chain(&dimensions)
        .any(|v| !v.is_finite())
    {
        return None;
    }
    let delta = [b[0] - a[0], b[1] - a[1]];
    if delta.iter().any(|value| !value.is_finite()) {
        return None;
    }
    let mut range: [f64; 2] = [0., 1.];
    for (p, q) in [
        (-delta[0], a[0]),
        (delta[0], dimensions[0] - a[0]),
        (-delta[1], a[1]),
        (delta[1], dimensions[1] - a[1]),
    ] {
        if p == 0. {
            if q < 0. {
                return None;
            }
            continue;
        }
        let t = q / p;
        if p < 0. {
            range[0] = range[0].max(t);
        } else {
            range[1] = range[1].min(t);
        }
        if range[0] > range[1] {
            return None;
        }
    }
    Some(range.map(|t| [a[0] + t * delta[0], a[1] + t * delta[1]]))
}
fn dashed(list: &imgui::DrawListMut<'_>, a: [f32; 2], b: [f32; 2], color: [f32; 4]) {
    let delta = [b[0] - a[0], b[1] - a[1]];
    let length = delta[0].hypot(delta[1]);
    if length == 0. {
        return;
    }
    let mut start = 0.;
    while start < length {
        let end = (start + 5.).min(length);
        let point = |t: f32| [a[0] + delta[0] * t / length, a[1] + delta[1] * t / length];
        list.add_line(point(start), point(end), color).build();
        start += 8.;
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn sheet_clips_projection_before_screen_conversion() {
        assert_eq!(
            clip_segment([-5., 5.], [15., 5.], [10., 10.]),
            Some([[0., 5.], [10., 5.]])
        );
        assert_eq!(clip_segment([-5., -1.], [15., -1.], [10., 10.]), None);
        assert_eq!(
            screen_point([0., 0.], [10., 20.], [110., 70.], [10., 5.]),
            [10., 70.]
        );
        assert_eq!(
            screen_point([10., 5.], [10., 20.], [110., 70.], [10., 5.]),
            [110., 20.]
        );
    }
}
