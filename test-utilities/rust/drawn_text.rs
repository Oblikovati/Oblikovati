// SPDX-License-Identifier: GPL-2.0-only
//! Read glyphs from actual completed draw packets; no widget state is substituted.
use imgui::{
    DrawCmd, DrawVert,
    internal::{RawCast, RawWrapper},
    sys,
};
use std::collections::{BTreeMap, BTreeSet};
type GlyphCoordinates = [u32; 4];
const QUAD_INDICES: usize = 6;
/// One drawn glyph in UI layout units, inside its clip rectangle.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) struct RenderedGlyph {
    pub letter: char,
    pub minimum: [f32; 2],
    pub maximum: [f32; 2],
    pub alpha: u8,
    pub color: [u8; 3],
}
/// A line of adjacent glyphs; buttons and menu rows are separate runs.
#[derive(Clone, Debug, Default, PartialEq)]
pub(crate) struct RenderedRun {
    pub text: String,
    pub minimum: [f32; 2],
    pub maximum: [f32; 2],
    pub alpha: u8,
    pub color: [u8; 3],
    pub list: usize,
}
/// A vertex of a non-rectangular shape (check marks, arrows), for controls without glyphs.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) struct RenderedMark {
    pub list: usize,
    pub position: [f32; 2],
    /// A mark's own colour: a window's dark rounded corner also counts as a mark.
    pub bright: bool,
}
/// Everything a scenario may read from one completed frame.
#[derive(Clone, Debug, Default, PartialEq)]
pub(crate) struct RenderedFrame {
    pub runs: Vec<RenderedRun>,
    pub marks: Vec<RenderedMark>,
    font_size: f32,
    popup_lists: BTreeSet<usize>,
}
pub(crate) fn run_is_checked(frame: &RenderedFrame, run: &RenderedRun) -> bool {
    frame.marks.iter().any(|mark| {
        if !mark.bright || mark.list != run.list || !same_check_row(frame, run, mark.position[1]) {
            return false;
        }
        let left = run.minimum[0] - mark.position[0];
        if left > 0. && left < frame.font_size * 2.5 {
            return true;
        }
        frame.is_popup_row(run)
            && mark.position[0] > run.maximum[0]
            && !frame.runs.iter().any(|other| {
                other.list == run.list
                    && other.minimum[0] > mark.position[0]
                    && other.minimum[0] - mark.position[0] < frame.font_size * 2.5
                    && same_check_row(frame, other, mark.position[1])
            })
    })
}
fn same_check_row(frame: &RenderedFrame, run: &RenderedRun, y: f32) -> bool {
    (y - run.center()[1]).abs() < frame.font_size * 0.5
}
fn is_popup_list(list: &imgui::DrawList) -> bool {
    // SAFETY: diagnostics run on the GUI owner thread while its ImGui context is live;
    // the borrowed context is not retained across another frame or context shutdown.
    let Some(context) = (unsafe { sys::igGetCurrentContext().as_ref() }) else {
        return false;
    };
    if context.Windows.Size <= 0 || context.Windows.Data.is_null() {
        return false;
    }
    // SAFETY: ImGui owns this non-null array and its recorded positive length. No frame
    // or window mutation occurs while the slice is borrowed, and it does not escape.
    let windows =
        unsafe { std::slice::from_raw_parts(context.Windows.Data, context.Windows.Size as usize) };
    // SAFETY: `list` is borrowed from this context's completed draw data and remains
    // live for this call; its raw address is used only for identity comparison.
    let draw = unsafe { list.raw() } as *const sys::ImDrawList;
    windows
        .iter()
        .filter_map(|window| {
            // SAFETY: entries are ImGui-owned window pointers from the live context;
            // null entries are skipped and no window is mutated during this borrow.
            unsafe { window.as_ref() }
        })
        .any(|window| std::ptr::eq(window.DrawList, draw) && belongs_to_popup(window))
}

fn belongs_to_popup(mut window: &sys::ImGuiWindow) -> bool {
    loop {
        if window.Flags & sys::ImGuiWindowFlags_Popup as i32 != 0 {
            return true;
        }
        if window.Flags & sys::ImGuiWindowFlags_ChildWindow as i32 == 0 {
            return false;
        }
        // SAFETY: each parent is an ImGui-owned window in the same live context.
        let Some(parent) = (unsafe { window.ParentWindow.as_ref() }) else {
            return false;
        };
        window = parent;
    }
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
pub(crate) fn rendered_frame() -> RenderedFrame {
    let raw =
        unsafe { sys::igGetDrawData().as_ref() }.expect("expected completed native GUI draws");
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    frame_of_draw_data(draws, &glyph_lookup(), current_font_size())
}
/// Check the completed draw list for a solid-colour line crossing a text run.
pub(crate) fn has_strike(run: &RenderedRun) -> bool {
    let raw =
        unsafe { sys::igGetDrawData().as_ref() }.expect("expected completed native GUI draws");
    let draws = unsafe { imgui::DrawData::from_raw(raw) };
    let io = unsafe { sys::igGetIO().as_ref() }.expect("expected live GUI input");
    let atlas = unsafe { io.Fonts.as_ref() }.expect("expected the font atlas");
    let Some(list) = draws.draw_lists().nth(run.list) else {
        return false;
    };
    let middle = run.center()[1];
    let points: Vec<_> = list
        .vtx_buffer()
        .iter()
        .filter(|vertex| {
            atlas.TexUvLines.iter().any(|line| {
                vertex.uv[1] == line.y && (vertex.uv[0] == line.x || vertex.uv[0] == line.z)
            }) && vertex.col[..3] == run.color
                && (vertex.pos[1] - middle).abs() <= 3.
                && vertex.pos[0] >= run.minimum[0] - 3.
                && vertex.pos[0] <= run.maximum[0] + 3.
        })
        .map(|vertex| vertex.pos[0])
        .collect();
    points
        .iter()
        .any(|left| points.iter().any(|right| right - left > 20.))
}
pub(crate) fn frame_of_draw_data(
    draws: &imgui::DrawData,
    lookup: &BTreeMap<GlyphCoordinates, char>,
    font_size: f32,
) -> RenderedFrame {
    let mut frame = RenderedFrame {
        font_size,
        ..Default::default()
    };
    for (index, list) in draws.draw_lists().enumerate() {
        if is_popup_list(list) {
            frame.popup_lists.insert(index);
        }
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
pub(crate) fn current_font_size() -> f32 {
    unsafe { sys::igGetCurrentContext().as_ref() }
        .expect("expected a live native GUI")
        .FontSize
}
pub(crate) fn glyph_lookup() -> BTreeMap<GlyphCoordinates, char> {
    let io = unsafe { sys::igGetIO().as_ref() }.expect("expected live GUI input");
    let atlas = unsafe { io.Fonts.as_ref() }.expect("expected the font atlas");
    if atlas.Fonts.Data.is_null() || atlas.Fonts.Size <= 0 {
        return BTreeMap::new();
    }
    // The current context owns all fonts used by the frame, including native editor faces.
    let fonts = unsafe { std::slice::from_raw_parts(atlas.Fonts.Data, atlas.Fonts.Size as usize) };
    fonts
        .iter()
        .flat_map(|&font| font_glyph_entries(unsafe { &*font }))
        .collect()
}

fn font_glyph_entries(font: &sys::ImFont) -> impl Iterator<Item = (GlyphCoordinates, char)> + '_ {
    // Each installed face has live glyphs until its atlas is replaced between frames.
    let glyphs = unsafe { std::slice::from_raw_parts(font.Glyphs.Data, font.Glyphs.Size as usize) };
    glyphs.iter().filter_map(|glyph| {
        let letter = char::from_u32(glyph.Codepoint())?;
        let uv = [glyph.U0, glyph.V0, glyph.U1, glyph.V1].map(f32::to_bits);
        Some((uv, letter))
    })
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
pub(crate) fn quad_glyph(
    first: u16,
    vertices: &[DrawVert],
    clip: [f32; 4],
    lookup: &BTreeMap<GlyphCoordinates, char>,
) -> Option<RenderedGlyph> {
    let start = vertices[first as usize];
    let opposite = vertices[first as usize + 2];
    let uv = [start.uv[0], start.uv[1], opposite.uv[0], opposite.uv[1]];
    let letter = glyph_for_uv(uv, lookup)?;
    let corners = &vertices[first as usize..first as usize + 4];
    let mut minimum = start;
    let mut maximum = opposite;
    for axis in 0..2 {
        minimum.pos[axis] = corners
            .iter()
            .map(|v| v.pos[axis])
            .fold(f32::INFINITY, f32::min);
        maximum.pos[axis] = corners
            .iter()
            .map(|v| v.pos[axis])
            .fold(f32::NEG_INFINITY, f32::max);
    }
    inside_clip(minimum, maximum, clip).then_some(RenderedGlyph {
        letter,
        minimum: minimum.pos,
        maximum: maximum.pos,
        alpha: start.col[3],
        color: [start.col[0], start.col[1], start.col[2]],
    })
}
pub(crate) fn glyph_for_uv(
    uv: [f32; 4],
    lookup: &BTreeMap<GlyphCoordinates, char>,
) -> Option<char> {
    if let Some(letter) = lookup.get(&uv.map(f32::to_bits)) {
        return Some(*letter);
    }
    lookup.iter().find_map(|(bounds, letter)| {
        let original = bounds.map(f32::from_bits);
        let contained =
            (0..2).all(|axis| uv[axis] >= original[axis] && uv[axis + 2] <= original[axis + 2]);
        let visible = (0..2)
            .all(|axis| uv[axis + 2] - uv[axis] >= (original[axis + 2] - original[axis]) * 0.85);
        (contained && visible).then_some(*letter)
    })
}
pub(crate) fn inside_clip(first: DrawVert, opposite: DrawVert, clip: [f32; 4]) -> bool {
    (0..2).all(|axis| {
        let length = opposite.pos[axis] - first.pos[axis];
        let visible = opposite.pos[axis].min(clip[axis + 2]) - first.pos[axis].max(clip[axis]);
        length > 0. && visible >= length * 0.85
    })
}
/// Check a named popup in its owning window; for example, ribbon overflow at a narrow width.
impl RenderedFrame {
    /// Check whether a rendered row came from a popup draw list.
    /// Example: `frame.is_popup_row(row)` distinguishes a menu option from its ribbon caption.
    pub(crate) fn is_popup_row(&self, row: &RenderedRun) -> bool {
        self.popup_lists.contains(&row.list)
    }
}
