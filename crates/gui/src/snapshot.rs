// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiBatch, GuiFrame, GuiTextureResources, GuiVertex};
use imgui::{DrawCmd, DrawCmdParams, DrawData, DrawList};
pub(crate) const FONT_TEXTURE_ID: usize = crate::GuiTextureKey::FONT.raw();
pub(crate) fn capture_frame(
    draws: &DrawData,
    extent: [u32; 2],
    resources: &GuiTextureResources,
) -> Result<GuiFrame, String> {
    let mut batches = Vec::new();
    let mut remapping = Vec::new();
    for list in draws.draw_lists() {
        let addressable = list
            .vtx_buffer()
            .len()
            .min(usize::from(imgui::DrawIdx::MAX) + 1);
        remapping.resize(addressable, u32::MAX);
        for command in list.commands() {
            if let Some(batch) = capture_command(list, command, draws, resources, &mut remapping)? {
                batches.push(batch);
            }
        }
    }
    Ok(GuiFrame {
        extent,
        batches,
        resources: resources.clone(),
    })
}
fn capture_command(
    list: &DrawList,
    command: DrawCmd,
    frame: &DrawData,
    resources: &GuiTextureResources,
    remapping: &mut [u32],
) -> Result<Option<GuiBatch>, String> {
    match command {
        DrawCmd::ResetRenderState => Ok(None),
        DrawCmd::RawCallback { .. } => {
            Err("GUI raw callback: expected an owned declarative draw command".into())
        }
        DrawCmd::Elements { count, cmd_params } => {
            capture_elements(list, count, cmd_params, frame, resources, remapping).map(Some)
        }
    }
}
fn capture_elements(
    list: &DrawList,
    count: usize,
    params: DrawCmdParams,
    frame: &DrawData,
    resources: &GuiTextureResources,
    remapping: &mut [u32],
) -> Result<GuiBatch, String> {
    let texture = resources.binding(params.texture_id.id())?;
    let span = command_index_span(params.idx_offset, count, list.idx_buffer().len())?;
    let selected = &list.idx_buffer()[span];
    let (vertices, indices) = compact_vertices(
        list.vtx_buffer(),
        selected,
        params.vtx_offset,
        frame,
        remapping,
    )?;
    Ok(GuiBatch {
        texture,
        vertices,
        indices,
        clip_rect: scaled_clip(params.clip_rect, frame),
    })
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
    source_vertices: &[imgui::DrawVert],
    source: &[imgui::DrawIdx],
    base: usize,
    frame: &DrawData,
    remapping: &mut [u32],
) -> Result<(Vec<GuiVertex>, Vec<u32>), String> {
    let mut vertices = Vec::new();
    let mut indices = Vec::with_capacity(source.len());
    for &index in source {
        let vertex = checked_vertex_reference(index, base, source_vertices.len())? as usize;
        let slot = &mut remapping[usize::from(index)];
        if *slot == u32::MAX {
            *slot = vertices.len() as u32;
            vertices.push(captured_vertex(&source_vertices[vertex], frame));
        }
        indices.push(*slot);
    }
    // Adjacent draw commands may share indices but own independent vertex arrays.
    for &index in source {
        remapping[usize::from(index)] = u32::MAX;
    }
    Ok((vertices, indices))
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

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn index_offsets_refuse_overflow_and_invalid_vertex_references() {
        assert_eq!(checked_vertex_reference(2, 3, 6).unwrap(), 5);
        assert_eq!(command_index_span(2, 3, 5).unwrap(), 2..5);
        assert!(command_index_span(2, 4, 5).is_err());
        assert!(command_index_span(usize::MAX, 1, usize::MAX).is_err());
        assert!(checked_vertex_reference(1, usize::MAX, usize::MAX).is_err());
        assert!(checked_vertex_reference(2, 3, 5).is_err());
    }

    #[test]
    fn adjacent_commands_remap_repeated_indices_independently() {
        let mut context = crate::test_context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [64., 64.];
        context.io_mut().display_framebuffer_scale = [2., 2.];
        context.fonts().build_rgba32_texture();
        context.frame();
        let frame = context.render();
        let source = [1., 2., 3., 4.].map(|x| imgui::DrawVert {
            pos: [x, 4.],
            uv: [x / 4., 0.],
            col: [0, 10, 20, 30],
        });
        let mut remapping = [u32::MAX; 4];
        let (first, first_indices) =
            compact_vertices(&source, &[1, 0, 1], 0, frame, &mut remapping).unwrap();
        let (second, second_indices) =
            compact_vertices(&source, &[0, 1, 0], 2, frame, &mut remapping).unwrap();
        assert_eq!(first_indices, [0, 1, 0]);
        assert_eq!(second_indices, [0, 1, 0]);
        assert_eq!(
            first
                .iter()
                .map(|vertex| vertex.position)
                .collect::<Vec<_>>(),
            [[4., 8.], [2., 8.]]
        );
        assert_eq!(
            second
                .iter()
                .map(|vertex| vertex.position)
                .collect::<Vec<_>>(),
            [[6., 8.], [8., 8.]]
        );
        assert_eq!(second[0].uv, [0.75, 0.]);
        assert_eq!(second[0].srgb, [0, 10, 20, 30]);
    }
}
