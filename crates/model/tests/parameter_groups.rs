// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};

fn grouped() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(9, "width", USER_PARAM, "1 mm")
        .unwrap();
    parameters.insert_text(2, "material", "steel").unwrap();
    parameters.insert_boolean(7, "vented", true).unwrap();
    parameters
        .add_group("z:first", "First", "addin.id")
        .unwrap();
    parameters.add_group("a:second", "", "").unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn group_keys_are_stable_and_order_is_creation_order() {
    let mut parameters = grouped();
    assert_eq!(parameters.groups()[0].key.0, "z:first");
    assert_eq!(parameters.groups()[1].display_name, "a:second");
    parameters
        .set_group_display_name("z:first", "Renamed")
        .unwrap();
    let group = parameters.group("z:first").unwrap();
    assert_eq!(group.key.0, "z:first");
    assert_eq!(group.display_name, "Renamed");
    assert_eq!(group.client_id, "addin.id");
    assert!(parameters.group("Renamed").is_none());
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn multi_memberships_are_idempotent_and_do_not_change_parameter_semantics() {
    let mut parameters = grouped();
    for id in [7, 2, 9, 9] {
        parameters.add_to_group(id, "a:second").unwrap();
    }
    parameters.add_to_group(9, "z:first").unwrap();
    assert_eq!(parameters.group_members("a:second").unwrap(), [9, 2, 7]);
    let keys: Vec<&str> = parameters
        .groups_of(9)
        .unwrap()
        .iter()
        .map(|key| key.0.as_ref())
        .collect();
    assert_eq!(keys, ["z:first", "a:second"]);
    parameters.remove_from_group(9, "a:second").unwrap();
    parameters.remove_from_group(9, "a:second").unwrap();
    assert_eq!(parameters.groups_of(9).unwrap().len(), 1);
    parameters.remove_from_all_groups(9).unwrap();
    assert!(parameters.groups_of(9).unwrap().is_empty());
    assert_eq!(parameters.ids(), &[9, 2, 7]);
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn invalid_group_operations_are_atomic() {
    let mut parameters = grouped();
    let before = parameters.groups().to_vec();
    assert!(parameters.add_group("", "x", "").is_err());
    assert!(parameters.add_group("z:first", "duplicate", "").is_err());
    let error = parameters
        .set_group_display_name("z:first", "")
        .unwrap_err()
        .to_string();
    assert!(error.contains("z:first") && error.contains("nonempty display name"));
    check_missing_groups(&mut parameters);
    assert_eq!(parameters.groups(), before);
    assert!(parameters.drain_changed().is_empty());
}

fn check_missing_groups(parameters: &mut ParameterCollection<u64>) {
    assert!(
        parameters
            .set_group_display_name("missing", "Name")
            .is_err()
    );
    assert!(parameters.delete_group("missing", true).is_err());
    assert!(parameters.add_to_group(9, "missing").is_err());
    assert!(parameters.remove_from_group(9, "missing").is_err());
    assert!(parameters.group_members("missing").is_err());
    assert!(parameters.add_to_group(99, "z:first").is_err());
    assert!(parameters.remove_from_group(99, "z:first").is_err());
    assert!(parameters.remove_from_all_groups(99).is_err());
    assert!(parameters.groups_of(99).is_err());
}

#[test]
fn plain_group_deletion_keeps_members_and_other_memberships() {
    let mut parameters = grouped();
    parameters.add_to_group(9, "z:first").unwrap();
    parameters.add_to_group(9, "a:second").unwrap();
    parameters.delete_group("z:first", false).unwrap();
    assert!(parameters.get(9).is_some());
    assert_eq!(parameters.group_members("a:second").unwrap(), [9]);
    assert_eq!(parameters.groups_of(9).unwrap().len(), 1);
    parameters.add_group("z:first", "", "").unwrap();
    assert!(parameters.group_members("z:first").unwrap().is_empty());
    assert_eq!(parameters.groups()[1].key.0, "z:first");
}

#[test]
fn parameter_removal_cleans_memberships_and_reused_ids_do_not_inherit_them() {
    let mut parameters = grouped();
    parameters.add_to_group(2, "z:first").unwrap();
    parameters.add_to_group(2, "a:second").unwrap();
    parameters.rename(2, "label").unwrap();
    assert_eq!(parameters.group_members("z:first").unwrap(), [2]);
    parameters.remove(2).unwrap();
    parameters.insert_text(2, "replacement", "new").unwrap();
    assert!(parameters.group_members("z:first").unwrap().is_empty());
    assert!(parameters.group_members("a:second").unwrap().is_empty());
}

#[test]
fn cascade_refuses_surviving_feature_dimensions_before_any_member_is_removed() {
    let mut parameters = grouped();
    parameters
        .insert_expression(5, "owned", MODEL_PARAM, "2 mm")
        .unwrap();
    parameters.add_to_group(9, "z:first").unwrap();
    parameters.add_to_group(5, "z:first").unwrap();
    parameters.drain_changed();
    let error = parameters
        .delete_group("z:first", true)
        .unwrap_err()
        .to_string();
    assert!(error.contains("feature dimension"));
    assert_eq!(parameters.ids(), &[9, 2, 7, 5]);
    assert_eq!(parameters.group_members("z:first").unwrap(), [9, 5]);
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn cascade_cleans_other_groups_and_preserves_unrelated_literals() {
    let mut parameters = grouped();
    parameters.add_to_group(9, "z:first").unwrap();
    parameters.add_to_group(2, "z:first").unwrap();
    parameters.add_to_group(2, "a:second").unwrap();
    parameters.delete_group("z:first", true).unwrap();
    assert_eq!(parameters.ids(), &[7]);
    assert!(parameters.boolean_value(7).unwrap());
    assert!(parameters.group_members("a:second").unwrap().is_empty());
    assert!(parameters.group("z:first").is_none());
}

#[test]
fn every_four_node_dag_and_group_subset_obeys_survivor_safety() {
    for edges in 0..64_u8 {
        for members in 0..16_u8 {
            check_cascade(edges, members);
        }
    }
}

const EDGES: [(u64, u64); 6] = [(0, 1), (0, 2), (1, 2), (0, 3), (1, 3), (2, 3)];

fn dag(edges: u8, members: u8) -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.add_group("G", "", "").unwrap();
    for id in 0..4 {
        let source = EDGES
            .iter()
            .enumerate()
            .filter(|(bit, edge)| edges & (1 << bit) != 0 && edge.1 == id)
            .map(|(_, edge)| format!("p{}", edge.0))
            .chain(["1".into()])
            .collect::<Vec<_>>()
            .join(" + ");
        parameters
            .insert_expression(id, &format!("p{id}"), USER_PARAM, &source)
            .unwrap();
        if members & (1 << id) != 0 {
            parameters.add_to_group(id, "G").unwrap();
        }
    }
    parameters.drain_changed();
    parameters
}

fn check_cascade(edges: u8, members: u8) {
    let mut parameters = dag(edges, members);
    let before = parameters.clone();
    let blocked = EDGES.iter().enumerate().any(|(bit, &(driver, reader))| {
        edges & (1 << bit) != 0 && members & (1 << driver) != 0 && members & (1 << reader) == 0
    });
    assert_eq!(
        parameters.delete_group("G", true).is_err(),
        blocked,
        "edges={edges} members={members}"
    );
    for id in 0..4 {
        let retained = blocked || members & (1 << id) == 0;
        assert_eq!(
            parameters.get(id),
            if retained { before.get(id) } else { None }
        );
    }
    assert_eq!(parameters.group("G").is_some(), blocked);
    if blocked {
        assert_eq!(parameters.groups(), before.groups());
    }
    assert!(parameters.drain_changed().is_empty());
}
