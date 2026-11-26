// SPDX-License-Identifier: GPL-2.0-only
use super::{
    LINE_COLOR,
    drawing_lines::{clip_segment, screen_point},
};
use oblikovati_model::drawing::DrawingAnnotationDefinition;

pub(super) fn draw(
    ui: &imgui::Ui,
    list: &imgui::DrawListMut<'_>,
    annotations: &[DrawingAnnotationDefinition],
    [minimum, maximum]: [[f32; 2]; 2],
    dimensions: [f64; 2],
) {
    let position = |point| screen_point(point, minimum, maximum, dimensions);
    list.with_clip_rect_intersect(minimum, maximum, || {
        for annotation in annotations {
            if let Some(note) = annotation.note_presentation() {
                draw_segments(list, note.leader_segments_mm, position, dimensions);
                let origin = position(note.anchor_mm);
                if origin.iter().all(|coordinate| coordinate.is_finite()) {
                    list.add_text(origin, LINE_COLOR, note.text);
                }
            }
            if let Some(tag) = annotation.revision_tag_presentation() {
                draw_segments(list, tag.segments_mm, position, dimensions);
                let center = position(tag.text_center_mm);
                if center.iter().all(|coordinate| coordinate.is_finite()) {
                    let size = ui.calc_text_size(tag.text);
                    let origin =
                        std::array::from_fn::<_, 2, _>(|axis| center[axis] - size[axis] * 0.5);
                    list.add_text(origin, LINE_COLOR, tag.text);
                }
            }
        }
    });
}

fn draw_segments(
    list: &imgui::DrawListMut<'_>,
    segments: impl IntoIterator<Item = [[f64; 2]; 2]>,
    position: impl Fn([f64; 2]) -> [f32; 2],
    dimensions: [f64; 2],
) {
    for [from, to] in segments {
        if let Some([from, to]) = clip_segment(from, to, dimensions) {
            list.add_line(position(from), position(to), LINE_COLOR)
                .build();
        }
    }
}
