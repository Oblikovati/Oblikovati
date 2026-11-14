// SPDX-License-Identifier: GPL-2.0-only
//! Inspect the icon quads submitted by the native chooser.
use super::*;
use imgui::{DrawCmd, internal::RawCast, sys};

pub(super) fn template_icon_size(world: &ApplicationWorld, label: &str) -> [f32; 2] {
    let title = world.run(label).center();
    let raw = unsafe { sys::igGetDrawData().as_ref() }.expect("expected completed GUI draws");
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    draws
        .draw_lists()
        .flat_map(|list| {
            list.commands()
                .filter_map(move |command| icon_bounds(list, command))
        })
        .filter(|(low, high)| {
            let center = [(low[0] + high[0]) * 0.5, (low[1] + high[1]) * 0.5];
            (center[0] - title[0]).abs() < 100.
                && center[1] <= title[1] + 10.
                && title[1] - center[1] < 100.
        })
        .min_by(|(a, _), (b, _)| (a[1] - title[1]).abs().total_cmp(&(b[1] - title[1]).abs()))
        .map(|(low, high)| [high[0] - low[0], high[1] - low[1]])
        .expect("expected an image near the template caption")
}
fn icon_bounds(list: &imgui::DrawList, command: DrawCmd) -> Option<([f32; 2], [f32; 2])> {
    let DrawCmd::Elements { count, cmd_params } = command else {
        return None;
    };
    if cmd_params.texture_id.id() == oblikovati_viewport::GuiTextureKey::FONT.raw() {
        return None;
    }
    let indices = &list.idx_buffer()[cmd_params.idx_offset..cmd_params.idx_offset + count];
    let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
    let mut low = [f32::INFINITY; 2];
    let mut high = [f32::NEG_INFINITY; 2];
    for index in indices {
        let position = vertices[*index as usize].pos;
        for axis in 0..2 {
            low[axis] = low[axis].min(position[axis]);
            high[axis] = high[axis].max(position[axis]);
        }
    }
    (low[0].is_finite() && low[1].is_finite()).then_some((low, high))
}
