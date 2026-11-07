// SPDX-License-Identifier: GPL-2.0-only
//! Read glyphs from actual completed draw packets; no widget state is substituted.
use imgui::{DrawCmd, DrawVert, internal::RawCast, sys};
use std::{collections::BTreeMap, ffi::CString};
type GlyphCoordinates = [u32; 4];
const QUAD_INDICES: usize = 6;
/// One drawn glyph in UI layout units, inside its clip rectangle.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(super) struct RenderedGlyph {
    pub letter: char,
    pub minimum: [f32; 2],
    pub maximum: [f32; 2],
    pub alpha: u8,
    pub color: [u8; 3],
}
/// A line of adjacent glyphs; buttons and menu rows are separate runs.
#[derive(Clone, Debug, Default, PartialEq)]
pub(super) struct RenderedRun {
    pub text: String,
    pub minimum: [f32; 2],
    pub maximum: [f32; 2],
    pub alpha: u8,
    pub color: [u8; 3],
    pub list: usize,
}
/// A vertex of a non-rectangular shape (check marks, arrows), for controls without glyphs.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(super) struct RenderedMark {
    pub list: usize,
    pub position: [f32; 2],
    /// A mark's own colour: a window's dark rounded corner also counts as a mark.
    pub bright: bool,
}
/// Everything a scenario may read from one completed frame.
#[derive(Clone, Debug, Default, PartialEq)]
pub(super) struct RenderedFrame {
    pub runs: Vec<RenderedRun>,
    pub marks: Vec<RenderedMark>,
}
impl RenderedRun {
    pub fn center(&self) -> [f32; 2] {
        std::array::from_fn(|axis| (self.minimum[axis] + self.maximum[axis]) * 0.5)
    }
    fn starts(glyph: RenderedGlyph, list: usize) -> Self {
        Self {
            list,
            text: glyph.letter.to_string(),
            minimum: glyph.minimum,
            maximum: glyph.maximum,
            alpha: glyph.alpha,
            color: glyph.color,
        }
    }
    fn continues(&self, glyph: RenderedGlyph, font_size: f32) -> bool {
        // Tight glyph quads vary with the letter, so limits come from the font size: a word gap
        // is well under it, and the padding between neighbouring buttons is over it.
        let gap = glyph.minimum[0] - self.maximum[0];
        (glyph.minimum[1] - self.minimum[1]).abs() < font_size * 0.7
            && (-1.0..font_size * 0.95).contains(&gap)
    }
    fn push(&mut self, glyph: RenderedGlyph) {
        self.text.push(glyph.letter);
        self.maximum[0] = self.maximum[0].max(glyph.maximum[0]);
        self.maximum[1] = self.maximum[1].max(glyph.maximum[1]);
        self.minimum[1] = self.minimum[1].min(glyph.minimum[1]);
    }
}
/// The completed frame in draw order, so the last matching run is topmost.
pub(super) fn rendered_frame() -> RenderedFrame {
    let raw =
        unsafe { sys::igGetDrawData().as_ref() }.expect("expected completed native GUI draws");
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    frame_of_draw_data(draws, &glyph_lookup(), current_font_size())
}
pub(super) fn frame_of_draw_data(
    draws: &imgui::DrawData,
    lookup: &BTreeMap<GlyphCoordinates, char>,
    font_size: f32,
) -> RenderedFrame {
    let mut frame = RenderedFrame::default();
    for (index, list) in draws.draw_lists().enumerate() {
        let first_of_list = frame.runs.len();
        let (glyphs, marks) = list_shapes(list, lookup);
        frame
            .marks
            .extend(marks.into_iter().map(|(position, bright)| RenderedMark {
                list: index,
                position,
                bright,
            }));
        for glyph in glyphs {
            match frame.runs[first_of_list..].last_mut() {
                Some(run) if run.continues(glyph, font_size) => run.push(glyph),
                _ => frame.runs.push(RenderedRun::starts(glyph, index)),
            }
        }
    }
    frame
}
pub(super) fn current_font_size() -> f32 {
    unsafe { sys::igGetCurrentContext().as_ref() }
        .expect("expected a live native GUI")
        .FontSize
}
pub(super) fn glyph_lookup() -> BTreeMap<GlyphCoordinates, char> {
    let context =
        unsafe { sys::igGetCurrentContext().as_ref() }.expect("expected a live native GUI");
    let font = unsafe { context.Font.as_ref() }.expect("expected the rendered font");
    let glyphs = unsafe { std::slice::from_raw_parts(font.Glyphs.Data, font.Glyphs.Size as usize) };
    glyphs
        .iter()
        .filter_map(|glyph| {
            let letter = char::from_u32(glyph.Codepoint())?;
            let uv = [glyph.U0, glyph.V0, glyph.U1, glyph.V1].map(f32::to_bits);
            Some((uv, letter))
        })
        .collect()
}
fn list_shapes(
    list: &imgui::DrawList,
    lookup: &BTreeMap<GlyphCoordinates, char>,
) -> (Vec<RenderedGlyph>, Vec<([f32; 2], bool)>) {
    let mut glyphs = Vec::new();
    let mut marks = Vec::new();
    for command in list.commands() {
        if let DrawCmd::Elements { count, cmd_params } = command {
            let indices = &list.idx_buffer()[cmd_params.idx_offset..cmd_params.idx_offset + count];
            let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
            let clip = cmd_params.clip_rect;
            append_command_shapes(&mut glyphs, &mut marks, indices, vertices, clip, lookup);
        }
    }
    (glyphs, marks)
}
fn append_command_shapes(
    glyphs: &mut Vec<RenderedGlyph>,
    marks: &mut Vec<([f32; 2], bool)>,
    indices: &[u16],
    vertices: &[DrawVert],
    clip: [f32; 4],
    lookup: &BTreeMap<GlyphCoordinates, char>,
) {
    // Triangles (tree arrows, check marks) share the buffer, so quads are not index-aligned.
    let mut at = 0;
    while at < indices.len() {
        if at + QUAD_INDICES > indices.len() || !is_quad(&indices[at..at + QUAD_INDICES]) {
            marks.push(mark_of(&vertices[indices[at] as usize]));
            at += 1;
            continue;
        }
        if let Some(glyph) = quad_glyph(indices[at], vertices, clip, lookup) {
            glyphs.push(glyph);
        } else if slanted_quad(indices[at], vertices) {
            marks.push(mark_of(&vertices[indices[at] as usize]));
        }
        at += QUAD_INDICES;
    }
}
const BRIGHT_MARK_CHANNEL: u8 = 90;
fn mark_of(vertex: &DrawVert) -> ([f32; 2], bool) {
    (
        vertex.pos,
        vertex.col[..3]
            .iter()
            .any(|channel| *channel > BRIGHT_MARK_CHANNEL),
    )
}
/// Anti-aliased strokes such as check marks are quads too, but never axis aligned.
fn slanted_quad(first: u16, vertices: &[DrawVert]) -> bool {
    let start = vertices[first as usize].pos;
    let next = vertices[first as usize + 1].pos;
    start[0] != next[0] && start[1] != next[1]
}
fn is_quad(indices: &[u16]) -> bool {
    let first = indices[0];
    indices[1] == first + 1
        && indices[2] == first + 2
        && indices[3] == first
        && indices[4] == first + 2
        && indices[5] == first + 3
}
fn quad_glyph(
    first: u16,
    vertices: &[DrawVert],
    clip: [f32; 4],
    lookup: &BTreeMap<GlyphCoordinates, char>,
) -> Option<RenderedGlyph> {
    let start = vertices[first as usize];
    let opposite = vertices[first as usize + 2];
    let uv = [start.uv[0], start.uv[1], opposite.uv[0], opposite.uv[1]].map(f32::to_bits);
    let letter = *lookup.get(&uv)?;
    inside_clip(start, opposite, clip).then_some(RenderedGlyph {
        letter,
        minimum: start.pos,
        maximum: opposite.pos,
        alpha: start.col[3],
        color: [start.col[0], start.col[1], start.col[2]],
    })
}
fn inside_clip(first: DrawVert, opposite: DrawVert, clip: [f32; 4]) -> bool {
    first.pos[0] >= clip[0]
        && first.pos[1] >= clip[1]
        && opposite.pos[0] <= clip[2]
        && opposite.pos[1] <= clip[3]
}
pub(super) fn window_geometry(title: &str) -> Option<([f32; 2], [f32; 2])> {
    let title = CString::new(title).unwrap();
    let window = unsafe { sys::igFindWindowByName(title.as_ptr()).as_ref() }?;
    window
        .Active
        .then_some(([window.Pos.x, window.Pos.y], [window.Size.x, window.Size.y]))
}
/// Framebuffer bounds for a named GUI window, including display density.
/// Example: `window_pixel_bounds("##navigation-bar")` for screenshot exclusion.
pub(super) fn window_pixel_bounds(title: &str) -> Option<[f32; 4]> {
    let (position, size) = window_geometry(title)?;
    let scale = unsafe { sys::igGetIO().as_ref() }?.DisplayFramebufferScale;
    Some([
        position[0] * scale.x,
        position[1] * scale.y,
        (position[0] + size[0]) * scale.x,
        (position[1] + size[1]) * scale.y,
    ])
}
/// Screen rectangles that belong to GUI windows, not to the model beneath them.
pub(super) fn window_occlusions() -> Vec<[f32; 4]> {
    let context =
        unsafe { sys::igGetCurrentContext().as_ref() }.expect("expected a live native GUI");
    let windows =
        unsafe { std::slice::from_raw_parts(context.Windows.Data, context.Windows.Size as usize) };
    let scale = context.IO.DisplayFramebufferScale;
    windows
        .iter()
        .filter_map(|window| {
            let window = unsafe { window.as_ref() }?;
            if !window.Active || window.Hidden {
                return None;
            }
            let bounds = window.OuterRectClipped;
            Some([
                bounds.Min.x * scale.x,
                bounds.Min.y * scale.y,
                bounds.Max.x * scale.x,
                bounds.Max.y * scale.y,
            ])
        })
        .collect()
}
#[cfg(test)]
#[path = "diagnostics_tests.rs"]
mod tests;
