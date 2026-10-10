// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{DERIVED_PARAM, USER_PARAM};
use oblikovati_model::{
    parameter::{DerivedTableHealth, ParameterCollection},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use std::collections::BTreeMap;

fn source() -> BTreeMap<String, Quantity> {
    BTreeMap::from([
        ("module".into(), Quantity::new(0.2, Dimension::Length)),
        ("teeth".into(), Quantity::new(24.0, Dimension::Unitless)),
    ])
}

fn table() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .add_derived_table(
            7,
            "gears.obk",
            &["module".into()],
            &source(),
            &BTreeMap::from([("module".into(), 1)]),
        )
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn creation_uses_supplied_ids_and_numeric_read_only_values() {
    let mut parameters = table();
    assert_eq!(parameters.ids(), [1]);
    assert_eq!(
        parameters.derived_parameter_id(7, "module").unwrap(),
        Some(1)
    );
    assert_eq!(parameters.derived_parameter_id(7, "missing").unwrap(), None);
    assert_eq!(parameters.get(1).unwrap().kind, DERIVED_PARAM);
    assert_eq!(
        parameters.get(1).unwrap().current_value().unwrap().value,
        0.2
    );
    assert_eq!(
        parameters.derived_table(7).unwrap().health,
        DerivedTableHealth::Healthy
    );
    assert!(parameters.set_expression(1, "2").is_err());
    parameters
        .add_derived_table(8, "empty.obk", &[], &BTreeMap::new(), &BTreeMap::new())
        .unwrap();
    assert!(parameters.derived_table(8).unwrap().linked.is_empty());
    assert!(parameters.derived_parameter_id(99, "module").is_err());
}

#[test]
fn kept_links_preserve_identity_local_names_and_metadata() {
    let mut parameters = table();
    parameters.rename(1, "local_module").unwrap();
    parameters.set_symmetric(1, 0.01).unwrap();
    let changed = BTreeMap::from([
        ("module".into(), Quantity::new(0.25, Dimension::Length)),
        ("teeth".into(), Quantity::new(30.0, Dimension::Unitless)),
    ]);
    parameters
        .set_derived_table_linked(
            7,
            &["teeth".into(), "module".into()],
            &changed,
            &BTreeMap::from([("teeth".into(), 9)]),
        )
        .unwrap();
    assert_eq!(parameters.ids(), [1, 9]);
    assert_eq!(
        parameters.derived_table(7).unwrap().linked,
        ["teeth", "module"]
    );
    assert_eq!(parameters.get(1).unwrap().name, "local_module");
    assert_eq!(parameters.get(1).unwrap().tolerance.upper, 0.01);
    assert_eq!(
        parameters.get(1).unwrap().current_value().unwrap().value,
        0.25
    );
    assert_eq!(
        parameters.derived_parameter_id(7, "module").unwrap(),
        Some(1)
    );
}

#[test]
fn unlink_invalidates_readers_and_removes_group_membership() {
    let mut parameters = table();
    parameters
        .insert_expression(2, "pitch", USER_PARAM, "module * 24")
        .unwrap();
    let saved = parameters.get(2).unwrap().last_good_value.clone();
    parameters.add_group("dimensions", "", "").unwrap();
    parameters.add_to_group(1, "dimensions").unwrap();
    parameters
        .set_derived_table_linked(
            7,
            &["teeth".into()],
            &source(),
            &BTreeMap::from([("teeth".into(), 3)]),
        )
        .unwrap();
    assert!(parameters.get(1).is_none());
    assert!(parameters.group_members("dimensions").unwrap().is_empty());
    assert!(parameters.get(2).unwrap().current_value().is_none());
    assert_eq!(parameters.get(2).unwrap().last_good_value, saved);
    assert_eq!(
        parameters.derived_parameter_id(7, "teeth").unwrap(),
        Some(3)
    );
    assert_eq!(parameters.derived_parameter_id(7, "module").unwrap(), None);
}

#[test]
fn pending_restored_links_can_gain_a_new_local_parameter() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .restore_derived_table(7, "gears.obk", &["module".into()], true)
        .unwrap();
    parameters
        .set_derived_table_linked(
            7,
            &["module".into()],
            &source(),
            &BTreeMap::from([("module".into(), 5)]),
        )
        .unwrap();
    assert_eq!(
        parameters.derived_parameter_id(7, "module").unwrap(),
        Some(5)
    );
    assert!(parameters.derived_table(7).unwrap().owned_by_feature);
    assert_eq!(
        parameters.derived_table(7).unwrap().health,
        DerivedTableHealth::Healthy
    );
}

#[test]
fn malformed_subsets_and_id_allocations_leave_the_collection_unchanged() {
    let mut parameters = table();
    for (linked, ids) in [
        (
            vec!["unknown".into()],
            BTreeMap::from([("unknown".into(), 2)]),
        ),
        (vec!["module".into(), "module".into()], BTreeMap::new()),
        (vec!["".into()], BTreeMap::new()),
        (vec!["teeth".into()], BTreeMap::new()),
        (vec!["module".into()], BTreeMap::from([("extra".into(), 2)])),
        (vec!["teeth".into()], BTreeMap::from([("teeth".into(), 1)])),
    ] {
        assert!(
            parameters
                .set_derived_table_linked(7, &linked, &source(), &ids)
                .is_err()
        );
        assert_eq!(parameters.ids(), [1]);
        assert_eq!(parameters.derived_table(7).unwrap().linked, ["module"]);
        assert!(parameters.drain_changed().is_empty());
    }
    assert!(
        parameters
            .set_derived_table_linked(99, &[], &source(), &BTreeMap::new())
            .is_err()
    );
}

#[test]
fn duplicate_ids_name_clashes_and_invalid_values_refuse_creation() {
    let mut parameters = table();
    let linked = vec!["module".into(), "teeth".into()];
    let ids = BTreeMap::from([("module".into(), 8), ("teeth".into(), 8)]);
    assert!(
        parameters
            .add_derived_table(9, "other", &linked, &source(), &ids)
            .is_err()
    );
    assert!(
        parameters
            .add_derived_table(
                9,
                "other",
                &["module".into()],
                &source(),
                &BTreeMap::from([("module".into(), 8)])
            )
            .is_err()
    );
    for quantity in [
        Quantity::new(f64::INFINITY, Dimension::Length),
        Quantity::new(1.0, Dimension::Text),
    ] {
        let invalid = BTreeMap::from([("teeth".into(), quantity)]);
        assert!(
            parameters
                .add_derived_table(
                    9,
                    "other",
                    &["teeth".into()],
                    &invalid,
                    &BTreeMap::from([("teeth".into(), 8)])
                )
                .is_err()
        );
        assert_eq!(parameters.derived_tables().len(), 1);
        assert_eq!(parameters.ids(), [1]);
    }
    assert!(
        parameters
            .add_derived_table(7, "other", &[], &source(), &BTreeMap::new())
            .is_err()
    );
}

#[test]
fn replacement_can_reuse_a_removed_local_name_with_a_new_identity() {
    let mut parameters = table();
    parameters.rename(1, "teeth").unwrap();
    parameters
        .set_derived_table_linked(
            7,
            &["teeth".into()],
            &source(),
            &BTreeMap::from([("teeth".into(), 9)]),
        )
        .unwrap();
    assert_eq!(parameters.find("teeth"), Some(9));
    assert!(parameters.get(1).is_none());
}

#[test]
fn owned_tables_require_owner_deletion_and_readers_are_invalidated() {
    let mut parameters = table();
    parameters
        .insert_expression(2, "pitch", USER_PARAM, "module*24")
        .unwrap();
    parameters.mark_derived_table_owned(7).unwrap();
    assert!(parameters.delete_derived_table(7).is_err());
    assert!(parameters.get(1).is_some());
    parameters.delete_derived_table_for_owner(7).unwrap();
    assert!(parameters.derived_tables().is_empty());
    assert!(parameters.get(1).is_none());
    assert!(parameters.get(2).unwrap().current_value().is_none());
    assert!(parameters.mark_derived_table_owned(7).is_err());
    assert!(parameters.delete_derived_table(7).is_err());
}

#[test]
fn deleting_user_and_empty_tables_retains_other_tables() {
    let mut parameters = table();
    parameters
        .restore_derived_table(9, "other", &["pending".into()], false)
        .unwrap();
    parameters.delete_derived_table(7).unwrap();
    assert_eq!(parameters.derived_tables().len(), 1);
    assert!(parameters.ids().is_empty());
    parameters.delete_derived_table(9).unwrap();
    assert!(parameters.derived_tables().is_empty());
}

#[test]
fn duplicate_new_ids_are_rejected_before_any_parameter_is_inserted() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    let linked = vec!["module".into(), "teeth".into()];
    let ids = BTreeMap::from([("module".into(), 4_u64), ("teeth".into(), 4)]);
    let error = parameters
        .add_derived_table(1, "source", &linked, &source(), &ids)
        .unwrap_err();
    assert!(error.to_string().contains("unique unused parameter IDs"));
    assert!(parameters.ids().is_empty());
    assert!(parameters.derived_tables().is_empty());
}
