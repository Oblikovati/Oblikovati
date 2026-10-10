// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{DERIVED_PARAM, USER_PARAM};
use oblikovati_model::{
    parameter::{DerivedTableHealth, ParameterCollection, ParameterHealth},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};
use std::collections::BTreeMap;

fn linked() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_value(
            1,
            "module",
            DERIVED_PARAM,
            Quantity::new(0.2, Dimension::Length),
        )
        .unwrap();
    parameters
        .insert_expression(2, "pitch", USER_PARAM, "module * 24")
        .unwrap();
    parameters
        .restore_derived_table(7, "gears.obk", &["module".into()], true)
        .unwrap();
    parameters.drain_changed();
    parameters
}

#[test]
fn restoration_preserves_identity_order_ownership_and_references() {
    let mut parameters = linked();
    parameters
        .restore_derived_table(2, "other.obk", &["later".into()], false)
        .unwrap();
    assert_eq!(
        parameters
            .derived_tables()
            .iter()
            .map(|table| table.id)
            .collect::<Vec<_>>(),
        [7, 2]
    );
    let table = parameters.derived_table(7).unwrap();
    assert!(table.owned_by_feature);
    assert_eq!(table.health, DerivedTableHealth::Unresolved);
    let reference = &parameters.derived_references(7).unwrap()[0];
    assert_eq!(
        (
            &*reference.parameter,
            &*reference.source_parameter,
            &*reference.source_document
        ),
        ("module", "module", "gears.obk")
    );
    assert!(parameters.derived_references(2).unwrap().is_empty());
    assert!(parameters.derived_references(99).is_err());
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn refresh_failure_blocks_readers_retains_values_and_recovers() {
    let mut parameters = linked();
    let source = BTreeMap::from([("module".into(), Quantity::new(0.25, Dimension::Length))]);
    parameters.sync_derived_table(7, Some(&source)).unwrap();
    assert_eq!(
        parameters.get(2).unwrap().current_value().unwrap().value,
        6.0
    );
    let previous = parameters.get(2).unwrap().last_good_value.clone();
    parameters.sync_derived_table(7, None).unwrap();
    assert_eq!(
        parameters.derived_table(7).unwrap().health,
        DerivedTableHealth::Unavailable
    );
    assert!(parameters.get(1).unwrap().current_value().is_none());
    assert!(parameters.get(2).unwrap().current_value().is_none());
    assert_eq!(parameters.get(2).unwrap().last_good_value, previous);
    parameters.rename(2, "pitch_new").unwrap();
    assert!(parameters.get(1).unwrap().current_value().is_none());
    parameters.sync_derived_table(7, Some(&source)).unwrap();
    assert_eq!(
        parameters.derived_table(7).unwrap().health,
        DerivedTableHealth::Healthy
    );
    assert_eq!(parameters.get(2).unwrap().health, ParameterHealth::Healthy);
    assert_eq!(
        parameters.get(2).unwrap().current_value().unwrap().value,
        6.0
    );
}

#[test]
fn missing_links_and_local_removal_are_visible_without_stale_references() {
    let mut parameters = linked();
    parameters
        .sync_derived_table(7, Some(&BTreeMap::new()))
        .unwrap();
    assert_eq!(
        parameters.derived_table(7).unwrap().health,
        DerivedTableHealth::Missing(vec!["module".into()])
    );
    parameters.rename(1, "local_module").unwrap();
    let reference = &parameters.derived_references(7).unwrap()[0];
    assert_eq!(
        (&*reference.parameter, &*reference.source_parameter),
        ("local_module", "module")
    );
    parameters.remove_for_owner(1).unwrap();
    assert!(parameters.derived_references(7).unwrap().is_empty());
    let source = BTreeMap::from([("module".into(), Quantity::new(0.25, Dimension::Length))]);
    parameters.sync_derived_table(7, Some(&source)).unwrap();
    assert_eq!(
        parameters.derived_table(7).unwrap().health,
        DerivedTableHealth::Missing(vec!["module".into()])
    );
}

#[test]
fn ambiguous_or_malformed_restoration_is_atomic() {
    let mut parameters = linked();
    for (id, source, names) in [
        (0, "x", vec![]),
        (-1, "x", vec![]),
        (7, "x", vec![]),
        (8, "", vec![]),
        (8, "x", vec!["".into()]),
        (8, "x", vec!["a".into(), "a".into()]),
        (8, "x", vec!["module".into()]),
        (8, "x", vec!["pitch".into()]),
    ] {
        let error = parameters
            .restore_derived_table(id, source, &names, false)
            .unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert_eq!(parameters.derived_tables().len(), 1);
    }
    assert!(parameters.drain_changed().is_empty());
}

#[test]
fn invalid_refresh_rolls_back_all_values_and_health() {
    let mut parameters = linked();
    parameters
        .insert_value(
            3,
            "teeth",
            DERIVED_PARAM,
            Quantity::new(24.0, Dimension::Unitless),
        )
        .unwrap();
    parameters
        .insert_value(
            4,
            "count",
            DERIVED_PARAM,
            Quantity::new(2.0, Dimension::Unitless),
        )
        .unwrap();
    parameters
        .restore_derived_table(
            8,
            "gear2.obk",
            &["count".into(), "teeth".into(), "pending".into()],
            false,
        )
        .unwrap();
    for quantity in [
        Quantity::new(f64::NAN, Dimension::Unitless),
        Quantity::new(1.0, Dimension::Text),
    ] {
        let before = parameters.get(3).unwrap().clone();
        let source = BTreeMap::from([
            ("count".into(), Quantity::new(9.0, Dimension::Unitless)),
            ("teeth".into(), quantity),
        ]);
        assert!(parameters.sync_derived_table(8, Some(&source)).is_err());
        assert_eq!(parameters.get(3).unwrap(), &before);
        assert_eq!(
            parameters.get(4).unwrap().current_value().unwrap().value,
            2.0
        );
        assert_eq!(
            parameters.derived_table(8).unwrap().health,
            DerivedTableHealth::Unresolved
        );
    }
    assert!(parameters.sync_derived_table(99, None).is_err());
    parameters
        .sync_derived_table(8, Some(&BTreeMap::new()))
        .unwrap();
    assert_eq!(
        parameters.derived_table(8).unwrap().health,
        DerivedTableHealth::Missing(vec!["count".into(), "teeth".into(), "pending".into()])
    );
}

#[test]
fn partial_source_failure_preserves_the_previous_complete_reader_result() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    for (id, name) in [(1, "a"), (2, "b")] {
        parameters
            .insert_value(
                id,
                name,
                DERIVED_PARAM,
                Quantity::new(1.0, Dimension::Unitless),
            )
            .unwrap();
    }
    parameters
        .insert_expression(3, "total", USER_PARAM, "a+b")
        .unwrap();
    parameters
        .restore_derived_table(1, "source", &["a".into(), "b".into()], false)
        .unwrap();
    let saved = parameters.get(3).unwrap().last_good_value.clone();
    let source = BTreeMap::from([("a".into(), Quantity::new(10.0, Dimension::Unitless))]);
    parameters.sync_derived_table(1, Some(&source)).unwrap();
    assert_eq!(parameters.get(3).unwrap().last_good_value, saved);
    assert!(parameters.get(3).unwrap().current_value().is_none());
}
