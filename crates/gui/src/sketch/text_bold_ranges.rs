// SPDX-License-Identifier: GPL-2.0-only
//! Edit only the selected Unicode scalars, preserving other character formatting.
use oblikovati_api::types::SketchTextRun;
use std::ops::Range;

/// Read the selection state, for example `selection_is_bold(&runs, 2..4)`.
pub(super) fn selection_is_bold(runs: &[SketchTextRun], selection: Range<u32>) -> bool {
    if selection.is_empty() {
        return false;
    }
    let mut covered = selection.start;
    for run in runs.iter().filter(|run| run.end > selection.start) {
        if run.start > covered || !run.bold {
            return false;
        }
        covered = run.end;
        if covered >= selection.end {
            return true;
        }
    }
    false
}

/// Toggle the selected range, for example `toggle_bold(&runs, 2..4)`.
pub(super) fn toggle_bold(runs: &[SketchTextRun], selection: Range<u32>) -> Vec<SketchTextRun> {
    if selection.is_empty() {
        return runs.to_vec();
    }
    let bold = !selection_is_bold(runs, selection.clone());
    let edges = span_edges(runs, selection.clone());
    let mut formatted = Vec::with_capacity(runs.len() + 2);
    let mut source = runs.iter().peekable();
    for span in edges.windows(2) {
        let start = span[0];
        while source.peek().is_some_and(|run| run.end <= start) {
            source.next();
        }
        let style = source.peek().filter(|run| run.start <= start).copied();
        let mut piece = style.cloned().unwrap_or_default();
        if selection.contains(&start) {
            piece.bold = bold;
        }
        append_style(&mut formatted, piece, start..span[1]);
    }
    formatted
}

fn span_edges(runs: &[SketchTextRun], selection: Range<u32>) -> Vec<u32> {
    let mut edges = Vec::with_capacity(2 + 2 * runs.len());
    edges.extend([selection.start, selection.end]);
    edges.extend(runs.iter().flat_map(|run| [run.start, run.end]));
    edges.sort_unstable();
    edges.dedup();
    edges
}

fn append_style(runs: &mut Vec<SketchTextRun>, mut style: SketchTextRun, range: Range<u32>) {
    style.start = 0;
    style.end = 0;
    if style == SketchTextRun::default() {
        return;
    }
    if let Some(prior) = runs.last_mut().filter(|prior| prior.end == range.start) {
        style.start = prior.start;
        style.end = prior.end;
        if style == *prior {
            prior.end = range.end;
            return;
        }
    }
    style.start = range.start;
    style.end = range.end;
    runs.push(style);
}

/// Keep formatting around a replacement, for example a typed suffix after a bold label.
pub(super) fn retain_unchanged_runs(
    runs: &[SketchTextRun],
    before: &str,
    after: &str,
) -> Vec<SketchTextRun> {
    let Some((old, new)) = replacement_bounds(before, after) else {
        return Vec::new();
    };
    let mut formatted = Vec::new();
    for run in runs.iter().take_while(|run| run.start < old.start) {
        append_style(
            &mut formatted,
            run.clone(),
            run.start..run.end.min(old.start),
        );
    }
    if !new.is_empty()
        && let Some(style) = replacement_style(runs, &old)
    {
        append_style(&mut formatted, style.clone(), new.clone());
    }
    for run in runs.iter().filter(|run| run.end > old.end) {
        let start = new.end + (run.start.max(old.end) - old.end);
        let end = new.end + (run.end - old.end);
        append_style(&mut formatted, run.clone(), start..end);
    }
    formatted
}

fn replacement_style<'a>(runs: &'a [SketchTextRun], old: &Range<u32>) -> Option<&'a SketchTextRun> {
    if !old.is_empty() || old.start == 0 {
        return runs
            .iter()
            .find(|run| run.start <= old.start && run.end > old.start);
    }
    runs.iter()
        .find(|run| run.start < old.start && run.end >= old.start)
}

fn replacement_bounds(before: &str, after: &str) -> Option<(Range<u32>, Range<u32>)> {
    let old_len = u32::try_from(before.chars().count()).ok()?;
    let new_len = u32::try_from(after.chars().count()).ok()?;
    let shared = old_len.min(new_len);
    let prefix = before
        .chars()
        .zip(after.chars())
        .take_while(|(a, b)| a == b)
        .count() as u32;
    let suffix = before
        .chars()
        .rev()
        .zip(after.chars().rev())
        .take((shared - prefix) as usize)
        .take_while(|(a, b)| a == b)
        .count() as u32;
    Some((prefix..old_len - suffix, prefix..new_len - suffix))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn toggling_mixed_selection_preserves_existing_style_and_gaps() {
        let original = vec![SketchTextRun {
            start: 1,
            end: 4,
            italic: true,
            ..Default::default()
        }];
        let updated = toggle_bold(&original, 2..6);
        assert!(selection_is_bold(&updated, 2..6));
        assert!(!selection_is_bold(&updated, 1..6));
        assert_eq!(
            updated
                .iter()
                .map(|run| (run.start, run.end, run.bold, run.italic))
                .collect::<Vec<_>>(),
            [(1, 2, false, true), (2, 4, true, true), (4, 6, true, false)]
        );
        assert_eq!(toggle_bold(&updated, 2..6), original);
    }

    #[test]
    fn empty_selection_and_adjacent_ranges_have_defined_highlight() {
        let first = SketchTextRun {
            start: 0,
            end: 1,
            bold: true,
            ..Default::default()
        };
        let second = SketchTextRun {
            start: 1,
            end: 2,
            italic: true,
            ..first.clone()
        };
        let existing = [first, second];
        assert!(selection_is_bold(&existing, 0..2));
        assert!(!selection_is_bold(&existing, 1..1));
        assert_eq!(toggle_bold(&existing, 1..1), existing);
        assert_eq!(
            toggle_bold(&existing, 0..2),
            [SketchTextRun {
                start: 1,
                end: 2,
                italic: true,
                ..Default::default()
            }]
        );
    }

    #[test]
    fn typing_after_a_bold_label_extends_the_style_and_retains_unicode_bounds() {
        let run = SketchTextRun {
            start: 0,
            end: 2,
            bold: true,
            ..Default::default()
        };
        let grown = retain_unchanged_runs(&[run.clone()], "a界", "a界!");
        assert_eq!(
            grown,
            [SketchTextRun {
                end: 3,
                ..run.clone()
            }]
        );
        assert_eq!(retain_unchanged_runs(&grown, "a界!", "界!"), [run]);
    }
}
