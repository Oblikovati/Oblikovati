// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn initial() -> DocumentUnitsInfo {
    DocumentUnitsInfo {
        length_unit: "mm".into(),
        angle_unit: "deg".into(),
        mass_unit: "kg".into(),
        time_unit: "s".into(),
        length_display_format: "decimal".into(),
        ..Default::default()
    }
}
#[test]
fn multiple_edits_form_one_partial_update_and_wait_for_success() {
    let before = initial();
    let mut draft = UnitDraft::new(&before);
    draft.value.length_unit = "in".into();
    draft.value.angle_display_precision = 3;
    let update = draft.submit(&before).unwrap();
    assert_eq!(
        update.length_unit.as_deref().map(String::as_str),
        Some("in")
    );
    assert_eq!(update.angle_display_precision.as_deref(), Some(&3));
    assert!(update.mass_unit.is_none());
    assert!(!draft.ready(&before));
    draft.acknowledge(Ok(()));
    assert!(!draft.dirty());
}
#[test]
fn failure_keeps_edits_and_allows_retry() {
    let before = initial();
    let mut draft = UnitDraft::new(&before);
    draft.value.length_unit = "in".into();
    assert!(draft.submit(&before).is_some());
    draft.acknowledge(Err("rejected".into()));
    assert_eq!(draft.error.as_deref(), Some("rejected"));
    assert!(draft.ready(&before));
    assert!(draft.submit(&before).is_some());
}
#[test]
fn stale_or_invalid_draft_does_not_submit() {
    let before = initial();
    let mut draft = UnitDraft::new(&before);
    draft.value.length_unit = "in".into();
    let mut external = before.clone();
    external.mass_unit = "lb".into();
    assert!(draft.submit(&external).is_none());
    draft.value.angle_display_precision = -1;
    assert!(draft.submit(&before).is_none());
}
#[test]
fn clean_draft_rebases_external_changes() {
    let before = initial();
    let mut draft = UnitDraft::new(&before);
    let mut external = before;
    external.length_unit = "in".into();
    draft.sync(&external);
    assert_eq!(draft.value.length_unit, "in");
    assert!(!draft.dirty());
}
