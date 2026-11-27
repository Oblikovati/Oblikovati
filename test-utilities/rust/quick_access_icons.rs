// SPDX-License-Identifier: GPL-2.0-only
//! Observe actual toolbar image quads without treating their labels as visible text (#4062).
use imgui::{DrawCmd, internal::RawCast, sys};

#[derive(Clone, Copy, Debug)]
pub(super) struct ToolbarImage {
    pub texture: usize,
    pub center: [f32; 2],
    pub maximum: [f32; 2],
    pub alpha: u8,
}

/// Read the standard icon row from completed draw data; example: `toolbar_images()` after a frame.
pub(super) fn toolbar_images() -> Vec<ToolbarImage> {
    let raw = unsafe { sys::igGetDrawData().as_ref() }.expect("expected completed GUI draws");
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    let scale = unsafe { sys::igGetIO().as_ref() }
        .expect("expected a live GUI context")
        .DisplayFramebufferScale;
    let brand = icon_texture(|glyph, _| glyph == "BrandMark");
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
            if Some(cmd_params.texture_id.id()) != brand
                && on_toolbar(center)
                && (14. ..=22.).contains(&(side[0] * scale.x))
                && (14. ..=22.).contains(&(side[1] * scale.y))
            {
                images.push(ToolbarImage {
                    texture: cmd_params.texture_id.id(),
                    center,
                    maximum: b,
                    alpha: vertices[indices[0] as usize].col[3],
                });
            }
        }
    }
    images.sort_by(|left, right| left.center[0].total_cmp(&right.center[0]));
    images
}

fn on_toolbar(center: [f32; 2]) -> bool {
    if let Some((position, size)) = super::drawn_windows::window_geometry("##quick-access-below") {
        return (0..2).all(|axis| {
            center[axis] >= position[axis] && center[axis] < position[axis] + size[axis]
        });
    }
    center[1] < 34.
}

/// Find a standard icon by its texture in the completed toolbar draw list.
/// Example: `standard_icon_center("Save")` when Save is visible.
pub(super) fn standard_icon_center(label: &str) -> Option<[f32; 2]> {
    standard_icon(label).map(|image| image.center)
}

pub(super) fn standard_icon(label: &str) -> Option<ToolbarImage> {
    let name = match label.replace(' ', "").as_str() {
        "New" => "new-part",
        "Open" => "open",
        "Save" => "save",
        "Undo" => "undo",
        "Redo" => "redo",
        "Home" => "home",
        "Projects" => "qat-projects",
        "Update" => "qat-update",
        "Select" => "qat-select",
        "MaterialBrowser" => "qat-material-browser",
        "Materials" => "qat-materials",
        "AppearanceBrowser" => "qat-appearance-browser",
        "Parameters" => "parameters",
        "Print" => "qat-print",
        "Extrude" => "extrude",
        _ => return None,
    };
    let texture = icon_texture(|_, asset| asset == name)?;
    toolbar_images()
        .into_iter()
        .find(|image| image.texture == texture)
}

fn icon_texture(matches: impl Fn(&str, &str) -> bool) -> Option<usize> {
    include_str!("../../crates/gui/src/icons/catalog.rs")
        .lines()
        .filter_map(|line| line.trim().split_once(" => \""))
        .position(|(glyph, asset)| matches(glyph, asset.trim_end_matches([',', '"'])))
        .map(|index| 2 + index * 2)
}
