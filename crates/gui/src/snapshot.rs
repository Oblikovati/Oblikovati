// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiBatch, GuiFrame, GuiVertex};
use imgui::{DrawCmd, DrawCmdParams, DrawData, DrawList};
use std::collections::BTreeMap;
pub(crate) const FONT_TEXTURE_ID: usize = 1;
pub(crate) fn capture_frame(
    draws: &DrawData,
    extent: [u32; 2],
    textures: &[crate::GuiTexture],
) -> Result<GuiFrame, String> {
    let mut batches = Vec::new();
    for list in draws.draw_lists() {
        for command in list.commands() {
            if let Some(batch) = capture_command(list, command, draws, textures)? {
                batches.push(batch);
            }
        }
    }
    Ok(GuiFrame {
        extent,
        textures: textures.to_vec(),
        batches,
    })
}
fn capture_command(
    list: &DrawList,
    command: DrawCmd,
    frame: &DrawData,
    textures: &[crate::GuiTexture],
) -> Result<Option<GuiBatch>, String> {
    match command {
        DrawCmd::ResetRenderState => Ok(None),
        DrawCmd::RawCallback { .. } => {
            Err("GUI raw callback: expected an owned declarative draw command".into())
        }
        DrawCmd::Elements { count, cmd_params } => {
            capture_elements(list, count, cmd_params, frame, textures).map(Some)
        }
    }
}
fn capture_elements(
    list: &DrawList,
    count: usize,
    params: DrawCmdParams,
    frame: &DrawData,
    textures: &[crate::GuiTexture],
) -> Result<GuiBatch, String> {
    require_texture_binding(params.texture_id.id(), textures)?;
    let span = command_index_span(params.idx_offset, count, list.idx_buffer().len())?;
    let selected = &list.idx_buffer()[span];
    let indices = capture_indices(selected, params.vtx_offset, list.vtx_buffer().len())?;
    let (vertices, indices) = compact_vertices(list, &indices, frame);
    Ok(GuiBatch {
        texture_id: params.texture_id.id(),
        vertices,
        indices,
        clip_rect: scaled_clip(params.clip_rect, frame),
    })
}
fn capture_indices(
    indices: &[imgui::DrawIdx],
    base: usize,
    vertex_count: usize,
) -> Result<Vec<u32>, String> {
    indices
        .iter()
        .map(|index| checked_vertex_reference(*index, base, vertex_count))
        .collect()
}
fn checked_vertex_reference(
    index: imgui::DrawIdx,
    base: usize,
    limit: usize,
) -> Result<u32, String> {
    let combined = base
        .checked_add(usize::from(index))
        .filter(|index| *index < limit)
        .ok_or_else(|| format!("GUI vertex {base}+{index}: expected index below {limit}"))?;
    u32::try_from(combined)
        .map_err(|cause| format!("GUI vertex {combined}: expected u32 index: {cause}"))
}
fn scaled_position(position: [f32; 2], frame: &DrawData) -> [f32; 2] {
    std::array::from_fn(|axis| {
        (position[axis] - frame.display_pos[axis]) * frame.framebuffer_scale[axis]
    })
}
fn scaled_clip(rect: [f32; 4], frame: &DrawData) -> [f32; 4] {
    let low = scaled_position([rect[0], rect[1]], frame);
    let high = scaled_position([rect[2], rect[3]], frame);
    [low[0], low[1], high[0], high[1]]
}

fn compact_vertices(
    list: &DrawList,
    source: &[u32],
    frame: &DrawData,
) -> (Vec<GuiVertex>, Vec<u32>) {
    let mut remapping = BTreeMap::new();
    let mut vertices = Vec::new();
    let mut indices = Vec::new();
    for index in source {
        let mapped = *remapping.entry(*index).or_insert_with(|| {
            let vertex = &list.vtx_buffer()[*index as usize];
            let mapped = vertices.len() as u32;
            vertices.push(captured_vertex(vertex, frame));
            mapped
        });
        indices.push(mapped);
    }
    (vertices, indices)
}

fn command_index_span(
    offset: usize,
    count: usize,
    limit: usize,
) -> Result<std::ops::Range<usize>, String> {
    let end = offset
        .checked_add(count)
        .filter(|end| *end <= limit)
        .ok_or_else(|| format!("GUI indices {offset}+{count}: expected range within {limit}"))?;
    Ok(offset..end)
}
fn captured_vertex(vertex: &imgui::DrawVert, frame: &DrawData) -> GuiVertex {
    GuiVertex {
        position: scaled_position(vertex.pos, frame),
        uv: vertex.uv,
        srgb: vertex.col,
    }
}

fn require_texture_binding(texture: usize, textures: &[crate::GuiTexture]) -> Result<(), String> {
    if texture == FONT_TEXTURE_ID || textures.iter().any(|candidate| candidate.id == texture) {
        return Ok(());
    }
    Err(format!(
        "GUI texture {texture}: expected font atlas {FONT_TEXTURE_ID} or an owned GUI texture"
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn index_offsets_refuse_overflow_and_invalid_vertex_references() {
        assert_eq!(capture_indices(&[0, 1, 2], 3, 6).unwrap(), [3, 4, 5]);
        require_texture_binding(FONT_TEXTURE_ID, &[]).unwrap();
        assert!(
            require_texture_binding(0, &[])
                .unwrap_err()
                .contains("GUI texture 0")
        );
        assert_eq!(command_index_span(2, 3, 5).unwrap(), 2..5);
        assert!(command_index_span(2, 4, 5).is_err());
        assert!(command_index_span(usize::MAX, 1, usize::MAX).is_err());
        assert!(capture_indices(&[1], usize::MAX, usize::MAX).is_err());
        assert!(capture_indices(&[2], 3, 5).is_err());
    }
}
