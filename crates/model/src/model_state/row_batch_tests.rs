// SPDX-License-Identifier: GPL-2.0-only
//! Atomic row changes retain member identities and copied values.
use super::*;
fn row(id: u64, name: &str, source: Option<u64>) -> ModelStateRowEdit {
    ModelStateRowEdit {
        id,
        name: name.into(),
        source,
    }
}
#[test]
fn batch_can_swap_names_insert_after_source_and_remove_a_member() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "First").unwrap();
    members.insert(3, "Second").unwrap();
    members.set_suppressed_for(2, 7, false, true).unwrap();
    members.activate(3).unwrap();
    members
        .replace_rows(&[
            row(1, "Primary", None),
            row(2, "Second", None),
            row(4, "Copy", Some(2)),
            row(3, "First", None),
        ])
        .unwrap();
    assert_eq!(
        members
            .list()
            .iter()
            .map(|member| member.id)
            .collect::<Vec<_>>(),
        vec![1, 2, 4, 3]
    );
    assert!(members.suppressed_for(4, 7, false).unwrap());
    assert_eq!(members.active().unwrap().id, 3);
    members
        .replace_rows(&[
            row(1, "Primary", None),
            row(2, "Second", None),
            row(4, "Copy", None),
        ])
        .unwrap();
    assert_eq!(members.active().unwrap().id, 1);
}
#[test]
fn bad_rows_are_refused_without_mutating_existing_members() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "First").unwrap();
    let before = members.clone();
    for rows in [
        vec![row(1, "Primary", None), row(2, "Primary", None)],
        vec![row(1, "Primary", None), row(3, "New", Some(9))],
        vec![row(1, "Primary", None), row(3, "New", Some(3))],
        vec![row(2, "First", None), row(1, "Primary", None)],
    ] {
        assert!(members.replace_rows(&rows).is_err());
        assert_eq!(members, before);
    }
}
