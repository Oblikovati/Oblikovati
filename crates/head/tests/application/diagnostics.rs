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
}
/// A line of adjacent glyphs; buttons and menu rows are separate runs.
#[derive(Clone, Debug, Default, PartialEq)]
pub(super) struct RenderedRun {
    pub text: String,
    pub minimum: [f32; 2],
    pub maximum: [f32; 2],
    pub alpha: u8,
}
impl RenderedRun {
    pub fn center(&self) -> [f32; 2] {
        std::array::from_fn(|axis| (self.minimum[axis] + self.maximum[axis]) * 0.5)
    }
    fn starts(glyph: RenderedGlyph) -> Self {
        Self {
            text: glyph.letter.to_string(),
            minimum: glyph.minimum,
            maximum: glyph.maximum,
            alpha: glyph.alpha,
        }
    }
    fn continues(&self, glyph: RenderedGlyph, font_size: f32) -> bool {
        // Tight glyph quads vary with the letter, so limits come from the font size: a word gap
        // is well under it, and the padding between neighbouring buttons is over it.
        let gap = glyph.minimum[0] - self.maximum[0];
        (glyph.minimum[1] - self.minimum[1]).abs() < font_size * 0.7
            && (-1.0..font_size * 0.85).contains(&gap)
    }
    fn push(&mut self, glyph: RenderedGlyph) {
        self.text.push(glyph.letter);
        self.maximum[0] = self.maximum[0].max(glyph.maximum[0]);
        self.maximum[1] = self.maximum[1].max(glyph.maximum[1]);
        self.minimum[1] = self.minimum[1].min(glyph.minimum[1]);
    }
}
/// Glyph runs of the completed frame in draw order, so the last match is topmost.
pub(super) fn rendered_runs() -> Vec<RenderedRun> {
    let raw =
        unsafe { sys::igGetDrawData().as_ref() }.expect("expected completed native GUI draws");
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    runs_of_draw_data(draws, &glyph_lookup(), current_font_size())
}
pub(super) fn runs_of_draw_data(
    draws: &imgui::DrawData,
    lookup: &BTreeMap<GlyphCoordinates, char>,
    font_size: f32,
) -> Vec<RenderedRun> {
    let mut runs: Vec<RenderedRun> = Vec::new();
    for list in draws.draw_lists() {
        let first_of_list = runs.len();
        for glyph in list_glyphs(list, lookup) {
            match runs[first_of_list..].last_mut() {
                Some(run) if run.continues(glyph, font_size) => run.push(glyph),
                _ => runs.push(RenderedRun::starts(glyph)),
            }
        }
    }
    runs
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
fn list_glyphs(
    list: &imgui::DrawList,
    lookup: &BTreeMap<GlyphCoordinates, char>,
) -> Vec<RenderedGlyph> {
    let mut glyphs = Vec::new();
    for command in list.commands() {
        if let DrawCmd::Elements { count, cmd_params } = command {
            let indices = &list.idx_buffer()[cmd_params.idx_offset..cmd_params.idx_offset + count];
            let vertices = &list.vtx_buffer()[cmd_params.vtx_offset..];
            append_command_glyphs(&mut glyphs, indices, vertices, cmd_params.clip_rect, lookup);
        }
    }
    glyphs
}
fn append_command_glyphs(
    glyphs: &mut Vec<RenderedGlyph>,
    indices: &[u16],
    vertices: &[DrawVert],
    clip: [f32; 4],
    lookup: &BTreeMap<GlyphCoordinates, char>,
) {
    // Triangles (tree arrows, check marks) share the buffer, so quads are not index-aligned.
    let mut at = 0;
    while at + QUAD_INDICES <= indices.len() {
        if !is_quad(&indices[at..at + QUAD_INDICES]) {
            at += 1;
            continue;
        }
        if let Some(glyph) = quad_glyph(indices[at], vertices, clip, lookup) {
            glyphs.push(glyph);
        }
        at += QUAD_INDICES;
    }
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
#[cfg(test)]
#[path = "diagnostics_tests.rs"]
mod tests;
