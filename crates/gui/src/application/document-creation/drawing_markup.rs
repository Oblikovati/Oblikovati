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
    let mm_per_pixel = f64::max(
        dimensions[0] / f64::from(maximum[0] - minimum[0]),
        dimensions[1] / f64::from(maximum[1] - minimum[1]),
    );
    list.with_clip_rect_intersect(minimum, maximum, || {
        for annotation in annotations {
            if let Some(datum) = annotation.datum_feature_presentation() {
                draw_segments(list, datum.frame_and_stem_mm, position, dimensions);
                let [a, b, c] = datum.triangle_mm.map(position);
                if a.into_iter().chain(b).chain(c).all(f32::is_finite) {
                    list.add_triangle(a, b, c, LINE_COLOR).filled(true).build();
                }
                center_text(ui, list, position(datum.text_center_mm), datum.letter);
            }
            if let Some(note) = annotation.note_presentation() {
                draw_segments(list, note.leader_segments_mm, position, dimensions);
                let origin = position(note.anchor_mm);
                if origin.iter().all(|coordinate| coordinate.is_finite()) {
                    list.add_text(origin, LINE_COLOR, note.text);
                }
            }
            if let Some(tag) = annotation.revision_tag_presentation() {
                draw_segments(list, tag.segments_mm, position, dimensions);
                center_text(ui, list, position(tag.text_center_mm), tag.text);
            }
            if let Some(cloud) = annotation.revision_cloud_presentation() {
                let segments = cloud.visible_segments(dimensions, mm_per_pixel);
                draw_segments(list, segments, position, dimensions);
                center_text(ui, list, position(cloud.label_center_mm()), cloud.tag);
            }
        }
    });
}

fn center_text(ui: &imgui::Ui, list: &imgui::DrawListMut<'_>, [x, y]: [f32; 2], text: &str) {
    if text.is_empty() || !x.is_finite() || !y.is_finite() {
        return;
    }
    let [width, height] = ui.calc_text_size(text);
    list.add_text([x - width * 0.5, y - height * 0.5], LINE_COLOR, text);
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
