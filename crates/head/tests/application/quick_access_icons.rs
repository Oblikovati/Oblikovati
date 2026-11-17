// SPDX-License-Identifier: GPL-2.0-only
//! Observe actual toolbar image quads without treating their labels as visible text (#4062).
use imgui::{DrawCmd, internal::RawCast, sys};

#[derive(Clone, Copy, Debug)]
pub(super) struct ToolbarImage {
    pub texture: usize,
    pub center: [f32; 2],
}

/// Read the standard icon row from completed draw data; example: `toolbar_images()` after a frame.
pub(super) fn toolbar_images() -> Vec<ToolbarImage> {
    let raw = unsafe { sys::igGetDrawData().as_ref() }.expect("expected completed GUI draws");
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    let mut images = Vec::new();
    for list in draws.draw_lists() {
        for command in list.commands() {
            let DrawCmd::Elements {
                count: 6,
                cmd_params,
            } = command
            else {
                continue;
            };
            let indices = &list.idx_buffer()[cmd_params.idx_offset..cmd_params.idx_offset + 6];
            let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
            let a = vertices[indices[0] as usize].pos;
            let b = vertices[indices[2] as usize].pos;
            let side = [b[0] - a[0], b[1] - a[1]];
            let center = [(a[0] + b[0]) / 2., (a[1] + b[1]) / 2.];
            if on_toolbar(center)
                && (14. ..=22.).contains(&side[0])
                && (14. ..=22.).contains(&side[1])
            {
                images.push(ToolbarImage {
                    texture: cmd_params.texture_id.id(),
                    center,
                });
            }
        }
    }
    images.sort_by(|left, right| left.center[0].total_cmp(&right.center[0]));
    images
}

fn on_toolbar(center: [f32; 2]) -> bool {
    if let Some((position, size)) = super::diagnostics::window_geometry("##quick-access-below") {
        return (0..2).all(|axis| {
            center[axis] >= position[axis] && center[axis] < position[axis] + size[axis]
        });
    }
    center[1] < 34.
}

/// Locate one of the three default icons by its rendered row position.
/// Example: `standard_icon_center("Save")` after all three defaults are visible.
pub(super) fn standard_icon_center(label: &str) -> Option<[f32; 2]> {
    let index = match label {
        "New" => 0,
        "Open" => 1,
        "Save" => 2,
        _ => return None,
    };
    let images = toolbar_images();
    if images.len() != 3
        || images[0].texture == images[1].texture
        || images[1].texture == images[2].texture
        || images[0].texture == images[2].texture
    {
        return None;
    }
    Some(images[index].center)
}
