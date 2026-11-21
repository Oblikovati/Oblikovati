// SPDX-License-Identifier: GPL-2.0-only
//! Feature table columns do not discard per-member suppression overrides.
use super::*;

#[test]
fn suppression_cells_edit_inactive_members_without_changing_active_selection() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Second").unwrap();
    members
        .set_table_columns(vec![ModelStateTableColumn::Suppression(7)])
        .unwrap();
    members.set_suppressed_for(2, 7, false, true).unwrap();
    assert_eq!(members.active().unwrap().id, 1);
    assert!(!members.suppressed_for(1, 7, false).unwrap());
    assert!(members.suppressed_for(2, 7, false).unwrap());
    assert_eq!(members.suppression_override(1, 7).unwrap(), None);
    assert_eq!(members.suppression_override(2, 7).unwrap(), Some(true));
    members.set_table_columns(vec![]).unwrap();
    assert!(members.suppressed_for(2, 7, false).unwrap());
    members
        .set_table_columns(vec![ModelStateTableColumn::Suppression(7)])
        .unwrap();
    assert!(members.suppressed_for(2, 7, false).unwrap());
}
#[test]
fn invalid_suppression_cells_and_column_lists_do_not_change_the_table() {
    let mut members = ModelStateMembers::default();
    members
        .set_table_columns(vec![ModelStateTableColumn::Suppression(7)])
        .unwrap();
    assert!(
        members
            .set_table_columns(vec![
                ModelStateTableColumn::Suppression(7),
                ModelStateTableColumn::Suppression(7)
            ])
            .is_err()
    );
    assert!(members.set_suppressed_for(99, 7, false, true).is_err());
    assert!(members.set_suppressed_for(1, 0, false, true).is_err());
    assert_eq!(
        members.table_columns(),
        &[ModelStateTableColumn::Suppression(7)]
    );
    assert!(!members.suppressed_for(1, 7, false).unwrap());
}
