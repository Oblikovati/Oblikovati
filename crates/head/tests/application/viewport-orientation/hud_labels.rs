// SPDX-License-Identifier: GPL-2.0-only
//! Check the orientation of submitted label glyphs and their rendered contrast.
use super::*;
use imgui::{DrawCmd, DrawData, DrawVert, internal::RawCast, sys};
use oblikovati_gui::GuiViewAxes;

const LABELS: [(&str, [f32; 3], [f32; 3]); 3] = [
    ("FRONT", [0., 0., 1.], [0., 1., 0.]),
    ("RIGHT", [1., 0., 0.], [0., 1., 0.]),
    ("TOP", [0., 1., 0.], [0., 0., -1.]),
];

struct Glyph {
    letter: char,
    corners: [[f32; 2]; 4],
}

impl Glyph {
    fn center(&self) -> [f32; 2] {
        std::array::from_fn(|axis| (self.corners[0][axis] + self.corners[2][axis]) / 2.)
    }
}

pub(super) fn verify(
    world: &mut ApplicationWorld,
    capture: &ViewportCapture,
    origin: [f32; 2],
    parts: GuiOrientationLayout,
    light: bool,
) {
    let axes =
        crate::view_orientation::view_axes(&super::super::super::viewport_pan::camera(world));
    let centers = LABELS.map(|(_, normal, _)| {
        let point = project(normal, &axes).map(|coordinate| coordinate * parts.half);
        at(at(origin, parts.center), point)
    });
    let glyphs = cube_glyphs();
    for (index, (title, _, up)) in LABELS.into_iter().enumerate() {
        let face: Vec<_> = glyphs
            .iter()
            .filter(|glyph| nearest_face(glyph.center(), centers) == index)
            .collect();
        let name: String = face.iter().map(|glyph| glyph.letter).collect();
        assert_eq!(name, title);
        let upward = project(up, &axes);
        for glyph in &face {
            let stem = std::array::from_fn(|axis| glyph.corners[0][axis] - glyph.corners[3][axis]);
            let alignment = dot(stem, upward) / (dot(stem, stem) * dot(upward, upward)).sqrt();
            assert!(
                alignment > 0.995,
                "{title} {} stem alignment {alignment}",
                glyph.letter
            );
        }
        assert!(
            face.iter()
                .filter(|glyph| visible_ink(capture, glyph, light))
                .count()
                >= 3,
            "expected high-contrast {title} glyphs in the {} theme",
            if light { "Light" } else { "Dark" }
        );
    }
}

fn project(vector: [f32; 3], axes: &GuiViewAxes) -> [f32; 2] {
    [dot(vector, axes.right), -dot(vector, axes.up)]
}

fn dot<const N: usize>(a: [f32; N], b: [f32; N]) -> f32 {
    a.into_iter().zip(b).map(|(a, b)| a * b).sum()
}

fn nearest_face(point: [f32; 2], centers: [[f32; 2]; 3]) -> usize {
    let distance = |center: [f32; 2]| (point[0] - center[0]).hypot(point[1] - center[1]);
    (0..centers.len())
        .min_by(|&a, &b| distance(centers[a]).total_cmp(&distance(centers[b])))
        .unwrap()
}

fn visible_ink(capture: &ViewportCapture, glyph: &Glyph, light: bool) -> bool {
    let [x, y] = glyph.center();
    let (minimum, maximum) = (-3..=3)
        .flat_map(|dy| (-3..=3).map(move |dx| [x + dx as f32, y + dy as f32]))
        .map(|point| image_pixel(capture, point)[1])
        .fold((u8::MAX, u8::MIN), |(low, high), value| {
            (low.min(value), high.max(value))
        });
    maximum - minimum >= 90 && if light { minimum < 105 } else { maximum > 155 }
}

fn cube_glyphs() -> Vec<Glyph> {
    let index = diagnostics::window_draw_list(CONTROL).expect("expected the ViewCube draw list");
    // ImGui owns the completed draw packet until the next frame begins.
    let raw = unsafe { sys::igGetDrawData().as_ref() }.expect("expected a completed GUI frame");
    let draws = unsafe { DrawData::from_raw(raw) };
    let list = draws.draw_lists().nth(index).unwrap();
    let lookup = diagnostics::glyph_lookup();
    list.commands()
        .flat_map(|command| {
            let DrawCmd::Elements { count, cmd_params } = command else {
                return Vec::new();
            };
            let indices = &list.idx_buffer()[cmd_params.idx_offset..cmd_params.idx_offset + count];
            let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
            indices
                .windows(6)
                .filter_map(|indices| {
                    let corners = glyph_vertices(indices, vertices)?;
                    let [a, _, b, _] = corners;
                    let letter =
                        diagnostics::glyph_for_uv([a.uv[0], a.uv[1], b.uv[0], b.uv[1]], &lookup)?;
                    Some(Glyph {
                        letter,
                        corners: corners.map(|vertex| vertex.pos),
                    })
                })
                .collect()
        })
        .collect()
}

fn glyph_vertices(indices: &[u16], vertices: &[DrawVert]) -> Option<[DrawVert; 4]> {
    let first = indices[0];
    let [one, two, three] = [
        first.checked_add(1)?,
        first.checked_add(2)?,
        first.checked_add(3)?,
    ];
    if indices != [first, one, two, first, two, three] {
        return None;
    }
    Some([first, one, two, three].map(|index| vertices[index as usize]))
}
