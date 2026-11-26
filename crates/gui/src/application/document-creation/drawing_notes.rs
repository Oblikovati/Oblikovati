// SPDX-License-Identifier: GPL-2.0-only
use super::{
    LINE_COLOR,
    drawing_lines::{clip_segment, screen_point},
};
use oblikovati_model::drawing::DrawingAnnotationDefinition;

pub(super) fn draw(
    list: &imgui::DrawListMut<'_>,
    annotations: &[DrawingAnnotationDefinition],
    [minimum, maximum]: [[f32; 2]; 2],
    dimensions: [f64; 2],
) {
    let position = |point| screen_point(point, minimum, maximum, dimensions);
    list.with_clip_rect_intersect(minimum, maximum, || {
        for note in annotations
            .iter()
            .filter_map(DrawingAnnotationDefinition::note_presentation)
        {
            for [from, to] in note.leader_segments_mm {
                if let Some([from, to]) = clip_segment(from, to, dimensions) {
                    list.add_line(position(from), position(to), LINE_COLOR)
                        .build();
                }
            }
            let origin = position(note.anchor_mm);
            if origin.iter().all(|coordinate| coordinate.is_finite()) {
                list.add_text(origin, LINE_COLOR, note.text);
            }
        }
    });
}
