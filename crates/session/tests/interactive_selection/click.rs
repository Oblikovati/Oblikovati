// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::selection::SelectionClickMode::{self, Replace, Toggle};
fn paired_selection() -> SelectionSet<AuthoredPick> {
    let mut selected = SelectionSet::default();
    selected.add(pick(1, 1, Edge));
    selected.add(pick(1, 2, Face));
    selected
}
#[test]
fn shift_control_and_both_modifiers_toggle_while_a_plain_click_replaces() {
    for (shift, control, expected) in [
        (false, false, Replace),
        (true, false, Toggle),
        (false, true, Toggle),
        (true, true, Toggle),
    ] {
        assert_eq!(SelectionClickMode::from_modifiers(shift, control), expected);
    }
}
#[test]
fn plain_click_replaces_and_repeat_click_requests_the_same_go_refresh() {
    let mut selected = paired_selection();
    assert_eq!(
        selected.apply_click(Some(pick(1, 1, Edge)), Replace, false),
        Some(1)
    );
    assert_eq!(selected.items(), &[pick(1, 1, Edge)]);
    assert_eq!(
        selected.apply_click(Some(pick(1, 1, Edge)), Replace, false),
        Some(1)
    );
    assert_eq!(selected.items(), &[pick(1, 1, Edge)]);
}
#[test]
fn modified_click_removes_one_identity_and_appends_a_new_identity_in_order() {
    let mut selected = paired_selection();
    let edge = pick(1, 1, Edge);
    assert_eq!(selected.apply_click(Some(edge), Toggle, false), Some(1));
    assert_eq!(selected.items(), &[pick(1, 2, Face)]);
    assert_eq!(selected.apply_click(Some(edge), Toggle, false), Some(2));
    assert_eq!(selected.items(), &[pick(1, 2, Face), edge]);
}
#[test]
fn modified_empty_click_preserves_selection_plain_empty_click_clears_once() {
    let mut selected = paired_selection();
    assert_eq!(selected.apply_click(None, Toggle, false), None);
    assert_eq!(selected.items().len(), 2);
    assert_eq!(selected.apply_click(None, Replace, false), Some(0));
    assert_eq!(selected.apply_click(None, Replace, false), None);
    assert!(selected.items().is_empty());
}
#[test]
fn rejected_addition_reports_cleared_membership_and_preserves_filter() {
    let mut selected = paired_selection();
    selected.set_filter(SelectionFilter::None);
    assert_eq!(
        selected.apply_click(Some(pick(1, 3, Body)), Replace, false),
        Some(0)
    );
    assert!(selected.items().is_empty());
    assert_eq!(selected.filter(), &SelectionFilter::None);
    assert_eq!(
        selected.apply_click(Some(pick(1, 3, Body)), Replace, false),
        None
    );
}
#[test]
fn modified_addition_is_filtered_but_existing_membership_can_be_removed() {
    let mut selected = paired_selection();
    selected.set_filter(SelectionFilter::None);
    assert_eq!(
        selected.apply_click(Some(pick(1, 3, Body)), Toggle, false),
        None
    );
    assert_eq!(
        selected.apply_click(Some(pick(1, 1, Edge)), Toggle, false),
        Some(1)
    );
    assert_eq!(selected.items(), &[pick(1, 2, Face)]);
}
#[test]
fn active_tools_own_all_hit_and_miss_inputs_without_changing_selection() {
    let mut selected = paired_selection();
    let before = selected.items().to_vec();
    for mode in [Replace, Toggle] {
        for candidate in [None, Some(pick(1, 1, Edge)), Some(pick(1, 3, Body))] {
            assert_eq!(selected.apply_click(candidate, mode, true), None);
            assert_eq!(selected.items(), before);
        }
    }
}
