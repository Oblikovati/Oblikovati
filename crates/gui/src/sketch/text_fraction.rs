// SPDX-License-Identifier: GPL-2.0-only
//! Stack one selected literal while keeping its slash and Unicode source offsets.
use super::{GuiSketchTextDraft, character_ranges::replacement_bounds};
use imgui::Ui;
use oblikovati_api::types::SketchTextFraction;
use std::ops::Range;

pub(super) fn draw(ui: &Ui, draft: &mut GuiSketchTextDraft, selection: &Range<usize>) -> bool {
    ui.same_line();
    let ranges = draft
        .formatting
        .stacked_fractions
        .as_deref()
        .unwrap_or_default();
    let fields = draft
        .formatting
        .linked_fields
        .as_deref()
        .unwrap_or_default();
    let change =
        selected_fraction(&draft.content, selection, ranges).filter(|change| match change {
            FractionEdit::Add(range) => !fields
                .iter()
                .any(|field| (range.start..range.end).contains(&field.at)),
            FractionEdit::Remove(_) => true,
        });
    let _disabled = ui.begin_disabled(change.is_none());
    let active = matches!(change, Some(FractionEdit::Remove(_)));
    if !crate::widget_abi::toggle_button(ui, "Stack", [0.; 2], active) {
        return false;
    }
    let Some(change) = change else { return false };
    apply(draft, change);
    true
}

enum FractionEdit {
    Add(SketchTextFraction),
    Remove(usize),
}

fn selected_fraction(
    content: &str,
    selection: &Range<usize>,
    ranges: &[SketchTextFraction],
) -> Option<FractionEdit> {
    let selected = content.get(selection.clone())?;
    let start = u32::try_from(content.get(..selection.start)?.chars().count()).ok()?;
    let width = u32::try_from(selected.chars().count()).ok()?;
    let end = start.checked_add(width)?;
    let proposed = SketchTextFraction { start, end };
    if let Some(index) = ranges.iter().position(|existing| existing == &proposed) {
        return Some(FractionEdit::Remove(index));
    }
    if ranges
        .iter()
        .any(|range| range.start < end && start < range.end)
    {
        return None;
    }
    let local = SketchTextFraction {
        start: 0,
        end: width,
    };
    SketchTextFraction::validate(Some(selected), &[local]).ok()?;
    Some(FractionEdit::Add(proposed))
}

fn apply(draft: &mut GuiSketchTextDraft, change: FractionEdit) {
    let ranges = draft
        .formatting
        .stacked_fractions
        .get_or_insert_with(Vec::new);
    match change {
        FractionEdit::Add(range) => {
            let at = ranges.partition_point(|previous| previous.start < range.start);
            ranges.insert(at, range);
        }
        FractionEdit::Remove(index) => {
            ranges.remove(index);
        }
    }
    if ranges.is_empty() {
        draft.formatting.stacked_fractions = None;
    }
}

pub(super) fn retain_unchanged(draft: &mut GuiSketchTextDraft, before: &str) {
    let Some(ranges) = draft.formatting.stacked_fractions.as_mut() else {
        return;
    };
    let Some((old, new)) = replacement_bounds(before, &draft.content) else {
        draft.formatting.stacked_fractions = None;
        return;
    };
    // An edit inside a fraction returns the changed source to ordinary text.
    ranges.retain_mut(|fraction| retain_fraction(fraction, &old, &new));
    if ranges.is_empty() {
        draft.formatting.stacked_fractions = None;
    }
}

fn retain_fraction(fraction: &mut SketchTextFraction, old: &Range<u32>, new: &Range<u32>) -> bool {
    if fraction.end <= old.start {
        return true;
    }
    if fraction.start < old.end {
        return false;
    }
    fraction.start = new.end + (fraction.start - old.end);
    fraction.end = new.end + (fraction.end - old.end);
    true
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stacked_selection_uses_scalar_offsets_and_toggles_without_changing_the_text() {
        let mut draft = GuiSketchTextDraft {
            content: "⌀1/2".into(),
            ..Default::default()
        };
        assert!(selected_fraction(&draft.content, &(0..1), &[]).is_none());
        let edit = selected_fraction(&draft.content, &(3..6), &[]).unwrap();
        apply(&mut draft, edit);
        let stored = draft.formatting.stacked_fractions.as_deref().unwrap();
        assert_eq!(stored, [SketchTextFraction { start: 1, end: 4 }]);
        let toggle = selected_fraction(&draft.content, &(3..6), stored).unwrap();
        apply(&mut draft, toggle);
        assert!(draft.formatting.stacked_fractions.is_none());
        assert_eq!(draft.content, "⌀1/2");
    }

    #[test]
    fn changes_before_a_stack_shift_it_and_changes_inside_unstack_it() {
        let mut draft = GuiSketchTextDraft {
            content: "⌀1/2".into(),
            ..Default::default()
        };
        let edit = selected_fraction(&draft.content, &(3..6), &[]).unwrap();
        apply(&mut draft, edit);
        let before = std::mem::replace(&mut draft.content, "⌀a1/2".into());
        retain_unchanged(&mut draft, &before);
        assert_eq!(
            draft.formatting.stacked_fractions.as_deref().unwrap(),
            [SketchTextFraction { start: 2, end: 5 }]
        );
        let before = std::mem::replace(&mut draft.content, "⌀a3/2".into());
        retain_unchanged(&mut draft, &before);
        assert!(draft.formatting.stacked_fractions.is_none());
    }
}
