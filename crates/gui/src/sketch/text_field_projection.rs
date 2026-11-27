// SPDX-License-Identifier: GPL-2.0-only
//! Translate native text editor offsets to authored linked field markers.
use oblikovati_model::sketch::SketchTextFieldProjection;
use std::ops::Range;

pub(super) struct ProjectedFieldEdit<'a> {
    pub authored: Range<usize>,
    pub input: Range<usize>,
    pub replacement: &'a str,
}

pub(super) fn project_position(view: &SketchTextFieldProjection, byte: usize) -> usize {
    remap_boundary(view, byte, |field| (&field.authored, &field.display), false)
}

pub(super) fn authored_selection(
    view: &SketchTextFieldProjection,
    display: Range<usize>,
    toward_end: bool,
) -> Range<usize> {
    let start = authored_boundary(view, display.start, display.is_empty() && toward_end);
    let end = authored_boundary(view, display.end, toward_end || !display.is_empty());
    start..end
}

fn authored_boundary(view: &SketchTextFieldProjection, byte: usize, toward_end: bool) -> usize {
    remap_boundary(
        view,
        byte,
        |field| (&field.display, &field.authored),
        toward_end,
    )
}

fn remap_boundary(
    view: &SketchTextFieldProjection,
    byte: usize,
    intervals: impl Fn(&oblikovati_model::sketch::SketchTextFieldSpan) -> (&Range<usize>, &Range<usize>),
    toward_end: bool,
) -> usize {
    let index = view
        .fields
        .partition_point(|field| intervals(field).0.start <= byte);
    let Some(last) = index
        .checked_sub(1)
        .and_then(|index| view.fields.get(index))
    else {
        return byte;
    };
    let (before, after) = intervals(last);
    if byte >= before.end {
        return after.end + (byte - before.end);
    }
    if byte > before.start && toward_end {
        return after.end;
    }
    after.start
}

pub(super) fn projected_field_edit<'a>(
    view: &SketchTextFieldProjection,
    input: &'a str,
    previous: Range<usize>,
    caret: usize,
) -> Option<ProjectedFieldEdit<'a>> {
    if view.content == input {
        return None;
    }
    let (old, new) = inferred_byte_edit(&view.content, input, previous, caret)
        .unwrap_or_else(|| common_byte_edit(&view.content, input));
    let authored =
        authored_boundary(view, old.start, false)..authored_boundary(view, old.end, true);
    let displayed = project_position(view, authored.start)..project_position(view, authored.end);
    let input_range = displayed.start..new.end + (displayed.end - old.end);
    Some(ProjectedFieldEdit {
        authored,
        input: input_range,
        replacement: &input[new],
    })
}

pub(super) fn inferred_byte_edit(
    before: &str,
    after: &str,
    previous: Range<usize>,
    caret: usize,
) -> Option<(Range<usize>, Range<usize>)> {
    let start = previous.start.min(caret);
    let tail = after.get(caret..)?;
    let old_end = before.len().checked_sub(tail.len())?;
    if old_end < previous.end
        || before.get(..start)? != after.get(..start)?
        || before.get(old_end..)? != tail
    {
        return None;
    }
    Some((start..old_end, start..caret))
}

fn common_byte_edit(before: &str, after: &str) -> (Range<usize>, Range<usize>) {
    let prefix: usize = before
        .chars()
        .zip(after.chars())
        .take_while(|(a, b)| a == b)
        .map(|(ch, _)| ch.len_utf8())
        .sum();
    let suffix: usize = before[prefix..]
        .chars()
        .rev()
        .zip(after[prefix..].chars().rev())
        .take_while(|(a, b)| a == b)
        .map(|(ch, _)| ch.len_utf8())
        .sum();
    (prefix..before.len() - suffix, prefix..after.len() - suffix)
}

#[cfg(test)]
#[path = "text_field_projection_tests.rs"]
mod tests;
