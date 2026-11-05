// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{
    DERIVED_PARAM, MODEL_PARAM, ParameterKind, REFERENCE_PARAM, TABLE_PARAM, USER_PARAM,
};
use oblikovati_model::{
    parameter::{ParameterCollection, ParameterError, ParameterFailure, ParameterHealth},
    quantity::{Dimension, Quantity},
    units::UnitsOfMeasure,
};

fn collection() -> ParameterCollection<u64> {
    ParameterCollection::new(UnitsOfMeasure::default())
}
fn add(parameters: &mut ParameterCollection<u64>, id: u64, name: &str, source: &str) {
    parameters
        .insert_expression(id, name, USER_PARAM, source)
        .unwrap();
}
fn current(parameters: &ParameterCollection<u64>, id: u64) -> f64 {
    parameters.get(id).unwrap().current_value().unwrap().value
}

#[test]
fn edits_recompute_the_diamond_once_and_leave_independent_parameters_untouched() {
    let mut parameters = collection();
    for (id, name, source) in [
        (4, "base", "2 cm"),
        (3, "left", "2 * base"),
        (2, "right", "base + 1 cm"),
        (1, "join", "left + right"),
        (9, "alone", "99 cm"),
    ] {
        add(&mut parameters, id, name, source);
    }
    parameters.drain_changed();
    parameters.set_expression(4, "5 cm").unwrap();
    assert_eq!(current(&parameters, 1), 16.0);
    assert_eq!(current(&parameters, 9), 99.0);
    assert_eq!(parameters.drain_changed(), [4, 3, 2, 1]);
    assert!(parameters.drain_changed().is_empty());
    assert_eq!(parameters.readers(4).unwrap(), [3, 2]);
    assert_eq!(parameters.drivers(1).unwrap(), [3, 2]);
    assert_eq!(parameters.find("join"), Some(1));
}

#[test]
fn malformed_edits_are_atomic_and_failed_formulas_retain_last_good_values() {
    let mut parameters = collection();
    add(&mut parameters, 1, "width", "5 cm");
    add(&mut parameters, 2, "twice", "2 * width");
    parameters.drain_changed();
    let before = parameters.get(1).unwrap().clone();
    assert!(parameters.set_expression(1, "2 +").is_err());
    assert_eq!(parameters.get(1), Some(&before));
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(1, "1 / 0").unwrap();
    assert_eq!(
        parameters.get(1).unwrap().last_good_value,
        before.last_good_value
    );
    assert!(parameters.get(1).unwrap().current_value().is_none());
    assert!(matches!(
        parameters.get(2).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::BlockedBy(_))
    ));
    parameters
        .set_value(1, Quantity::new(8.0, Dimension::Length))
        .unwrap();
    assert_eq!(current(&parameters, 2), 16.0);
}

#[test]
fn cycle_edits_retain_source_detach_drivers_and_quarantine_readers() {
    let mut parameters = collection();
    add(&mut parameters, 1, "a", "2 cm");
    add(&mut parameters, 2, "b", "a + 1 cm");
    assert!(matches!(
        parameters.set_expression(1, "b + 1 cm"),
        Err(ParameterError::Dependency(_))
    ));
    assert_eq!(parameters.get(1).unwrap().source, "b + 1 cm");
    assert!(parameters.drivers(1).unwrap().is_empty());
    assert!(parameters.get(1).unwrap().current_value().is_none());
    assert!(parameters.get(2).unwrap().current_value().is_none());
    parameters.set_expression(1, "10 cm").unwrap();
    assert_eq!(current(&parameters, 2), 11.0);
}

#[test]
fn forward_references_recover_and_forward_cycles_become_health() {
    let mut parameters = collection();
    add(&mut parameters, 1, "dependent", "base * 2");
    assert!(matches!(
        parameters.get(1).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::Unresolved(_))
    ));
    add(&mut parameters, 2, "base", "5 cm");
    assert_eq!(current(&parameters, 1), 10.0);
    add(&mut parameters, 3, "first", "later + 1");
    add(&mut parameters, 4, "later", "first + 1");
    assert!(matches!(
        parameters.get(3).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::Cycle(_))
    ));
    assert!(parameters.get(4).unwrap().current_value().is_none());
    parameters.set_expression(3, "1").unwrap();
    assert_eq!(current(&parameters, 4), 2.0);
}

#[test]
fn read_only_values_update_only_through_the_owner_path() {
    let mut parameters = collection();
    for (id, kind) in [(1, REFERENCE_PARAM), (2, DERIVED_PARAM)] {
        parameters
            .insert_value(
                id,
                &format!("measure{id}"),
                kind,
                Quantity::new(3.0, Dimension::Length),
            )
            .unwrap();
        assert!(parameters.set_expression(id, "4 cm").is_err());
        assert!(
            parameters
                .set_value(id, Quantity::new(4.0, Dimension::Length))
                .is_err()
        );
        parameters
            .update_measured(id, Quantity::new(5.0, Dimension::Length))
            .unwrap();
        assert_eq!(current(&parameters, id), 5.0);
    }
    parameters
        .insert_expression(3, "table", TABLE_PARAM, "measure1 + measure2")
        .unwrap();
    assert_eq!(current(&parameters, 3), 10.0);
    assert!(
        parameters
            .update_measured(3, Quantity::new(1.0, Dimension::Length))
            .is_err()
    );
}

#[test]
fn delete_guard_names_readers_and_owner_delete_invalidates_the_full_tail() {
    let mut parameters = collection();
    parameters
        .insert_expression(1, "d0", MODEL_PARAM, "10 cm")
        .unwrap();
    add(&mut parameters, 2, "half", "d0 / 2");
    add(&mut parameters, 3, "sum", "d0 + half");
    let error = parameters.remove(1).unwrap_err().to_string();
    assert!(error.contains("half") && error.contains("owning feature"));
    assert_eq!(current(&parameters, 3), 15.0);
    parameters.remove_for_owner(1).unwrap();
    assert!(parameters.get(1).is_none());
    assert!(parameters.find("d0").is_none());
    assert!(parameters.get(2).unwrap().current_value().is_none());
    assert!(parameters.get(3).unwrap().current_value().is_none());
    add(&mut parameters, 4, "d0", "20 cm");
    assert_eq!(current(&parameters, 3), 30.0);
}

#[test]
fn insertion_validation_leaves_no_half_registered_parameter() {
    let mut parameters = collection();
    add(&mut parameters, 1, "x", "1");
    for (id, name, kind, source) in [
        (1, "other", USER_PARAM, "1"),
        (2, "x", USER_PARAM, "1"),
        (2, "", USER_PARAM, "1"),
        (2, "self", USER_PARAM, "self+1"),
        (2, "bad", USER_PARAM, "1+"),
        (2, "readonly", REFERENCE_PARAM, "1"),
    ] {
        assert!(
            parameters
                .insert_expression(id, name, kind, source)
                .is_err()
        );
        assert_eq!(parameters.ids(), [1]);
    }
    assert!(
        parameters
            .insert_value(
                2,
                "bad",
                ParameterKind(255),
                Quantity::new(1.0, Dimension::Unitless)
            )
            .is_err()
    );
    assert!(parameters.set_expression(999, "1").is_err());
    assert!(parameters.drivers(999).is_err());
    assert!(parameters.readers(999).is_err());
    assert!(parameters.remove(999).is_err());
    parameters.remove(1).unwrap();
    assert!(parameters.ids().is_empty());
}

#[test]
fn cloned_scaled_collections_are_independent_and_nonfinite_results_are_failed() {
    let units = UnitsOfMeasure::default().centered_on_length("µm").unwrap();
    let mut parameters = ParameterCollection::new(units);
    add(&mut parameters, 1, "width", "5 µm");
    add(&mut parameters, 2, "area", "width * width");
    let mut copy = parameters.clone();
    copy.set_expression(1, "10 µm").unwrap();
    assert_eq!(current(&parameters, 2), 25.0);
    assert_eq!(current(&copy, 2), 100.0);
    copy.set_expression(1, "sqrt(-1)").unwrap();
    assert!(matches!(
        copy.get(1).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::NonFinite(_))
    ));
    assert!(copy.get(2).unwrap().current_value().is_none());
}
