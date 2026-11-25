// SPDX-License-Identifier: GPL-2.0-only
//! Read the actual image rectangles submitted for the current native GUI frame.
use imgui::{DrawCmd, DrawData, internal::RawCast, sys};

type Rectangle = ([f32; 2], [f32; 2]);

pub(in super::super) fn ribbon_icon_quads(band_bottom: f32) -> Vec<Rectangle> {
    icon_quads_inside(([0., 58.], [f32::MAX, band_bottom]))
}

pub(in super::super) fn icon_quads_inside(bounds: Rectangle) -> Vec<Rectangle> {
    icon_images_inside(bounds)
        .into_iter()
        .map(|(rectangle, _)| rectangle)
        .collect()
}

pub(in super::super) fn icon_images_inside(bounds: Rectangle) -> Vec<(Rectangle, usize)> {
    // ImGui owns the completed draw packet until the next frame begins.
    let raw = unsafe { sys::igGetDrawData().as_ref() }.expect("expected a completed GUI frame");
    let draws = unsafe { DrawData::from_raw(raw) };
    let mut images = Vec::new();
    for list in draws.draw_lists() {
        for command in list.commands() {
            let DrawCmd::Elements { count, cmd_params } = command else {
                continue;
            };
            if cmd_params.texture_id.id() == oblikovati_gui::GuiTextureKey::FONT.raw() {
                continue;
            }
            let indices = &list.idx_buffer()[cmd_params.idx_offset..cmd_params.idx_offset + count];
            let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
            for quad in indices.as_chunks::<6>().0 {
                let rectangle = (
                    vertices[quad[0] as usize].pos,
                    vertices[quad[2] as usize].pos,
                );
                if is_icon_inside(rectangle, bounds) {
                    images.push((rectangle, cmd_params.texture_id.id()));
                }
            }
        }
    }
    images
}
fn is_icon_inside((first, opposite): Rectangle, (minimum, maximum): Rectangle) -> bool {
    let width = opposite[0] - first[0];
    let height = opposite[1] - first[1];
    first[0] >= minimum[0]
        && first[1] >= minimum[1]
        && opposite[0] <= maximum[0]
        && opposite[1] <= maximum[1]
        && width >= 15.
        && (width - height).abs() < 0.1
}
