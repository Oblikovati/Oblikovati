// SPDX-License-Identifier: GPL-2.0-only
//! Convert selected characters while keeping formatting attached to each source scalar.
use super::GuiSketchTextDraft;
use oblikovati_api::types::SketchTextRun;
use std::ops::Range;

/// Uppercase one byte selection, for example `uppercase_selection(&mut draft, 0..2)`.
pub(super) fn uppercase_selection(
    draft: &mut GuiSketchTextDraft,
    selection: Range<usize>,
) -> Option<Range<usize>> {
    let selected = draft.content.get(selection.clone())?;
    let (replacement, boundaries) = expanded_scalars(selected)?;
    if replacement == selected {
        return None;
    }
    let start = selection.start;
    let scalar_start = u32::try_from(draft.content[..start].chars().count()).ok()?;
    if let Some(runs) = &mut draft.formatting.runs {
        remap_runs(runs, scalar_start, &boundaries)?;
    }
    if let Some(fractions) = &mut draft.formatting.stacked_fractions {
        for fraction in fractions {
            fraction.start = expanded_position(fraction.start, scalar_start, &boundaries)?;
            fraction.end = expanded_position(fraction.end, scalar_start, &boundaries)?;
        }
    }
    draft.content.replace_range(selection, &replacement);
    Some(start..start + replacement.len())
}

fn expanded_scalars(selected: &str) -> Option<(String, Vec<u32>)> {
    let mut uppercase = String::with_capacity(selected.len());
    let mut boundaries = vec![0_u32];
    for scalar in selected.chars() {
        let expanded = scalar.to_uppercase();
        let length = u32::try_from(expanded.clone().count()).ok()?;
        boundaries.push(boundaries.last()?.checked_add(length)?);
        uppercase.extend(expanded);
    }
    Some((uppercase, boundaries))
}

fn remap_runs(runs: &mut [SketchTextRun], start: u32, boundaries: &[u32]) -> Option<()> {
    let positions = runs
        .iter()
        .map(|run| {
            Some((
                expanded_position(run.start, start, boundaries)?,
                expanded_position(run.end, start, boundaries)?,
            ))
        })
        .collect::<Option<Vec<_>>>()?;
    for (run, (first, last)) in runs.iter_mut().zip(positions) {
        run.start = first;
        run.end = last;
    }
    Some(())
}

fn expanded_position(position: u32, start: u32, boundaries: &[u32]) -> Option<u32> {
    let Some(relative) = position.checked_sub(start) else {
        return Some(position);
    };
    let relative_index = usize::try_from(relative).ok()?;
    if let Some(&expanded) = boundaries.get(relative_index) {
        return start.checked_add(expanded);
    }
    let length = u32::try_from(boundaries.len().checked_sub(1)?).ok()?;
    position.checked_add(boundaries.last()?.checked_sub(length)?)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn uppercase_preserves_each_format_range_across_unicode_expansion() {
        let mut draft = formatted_label();
        assert_eq!(uppercase_selection(&mut draft, 3..6), Some(3..6));
        assert_eq!(draft.content, "界SSAB");
        let [bold, italic, width] = draft.formatting.runs.as_deref().unwrap() else {
            panic!("expected three formatting runs");
        };
        assert_eq!((bold.start, bold.end, bold.bold), (1, 3, true));
        assert_eq!((italic.start, italic.end, italic.italic), (3, 4, true));
        assert_eq!(
            (width.start, width.end, width.width_percent.as_deref()),
            (4, 5, Some(&50.))
        );
    }

    #[test]
    fn empty_or_split_utf8_selection_leaves_draft_unchanged() {
        let original = formatted_label();
        let mut draft = original.clone();
        assert_eq!(uppercase_selection(&mut draft, 3..3), None);
        assert_eq!(uppercase_selection(&mut draft, 1..3), None);
        assert_eq!(draft, original);
    }

    fn formatted_label() -> GuiSketchTextDraft {
        let mut draft = GuiSketchTextDraft {
            content: "界ßaB".into(),
            ..Default::default()
        };
        draft.formatting.runs = Some(vec![
            SketchTextRun {
                start: 1,
                end: 2,
                bold: true,
                ..Default::default()
            },
            SketchTextRun {
                start: 2,
                end: 3,
                italic: true,
                ..Default::default()
            },
            SketchTextRun {
                start: 3,
                end: 4,
                width_percent: Some(Box::new(50.)),
                ..Default::default()
            },
        ]);
        draft
    }
}
