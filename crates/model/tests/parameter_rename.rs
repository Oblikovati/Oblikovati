// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
use oblikovati_model::{
    parameter::{ParameterCollection, ParameterFailure, ParameterHealth},
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

fn value(parameters: &ParameterCollection<u64>, id: u64) -> f64 {
    parameters.get(id).unwrap().current_value().unwrap().value
}

#[test]
fn rename_preserves_ids_edges_spacing_and_later_recomputation() {
    let mut parameters = collection();
    add(&mut parameters, 1, "width", "10 cm");
    add(&mut parameters, 2, "height", " 2 * width\t+ 5 mm + width ");
    parameters.drain_changed();
    parameters.rename(1, "span").unwrap();
    assert_eq!(
        parameters.get(2).unwrap().source,
        " 2 * span\t+ 5 mm + span "
    );
    assert_eq!(parameters.find("width"), None);
    assert_eq!(parameters.find("span"), Some(1));
    assert_eq!(parameters.drivers(2).unwrap(), [1]);
    assert_eq!(parameters.readers(1).unwrap(), [2]);
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(1, "20 cm").unwrap();
    assert_eq!(value(&parameters, 2), 60.5);
}

#[test]
fn function_names_and_identifiers_with_a_common_prefix_are_not_rewritten() {
    let mut parameters = collection();
    add(&mut parameters, 1, "sin", "2");
    add(&mut parameters, 2, "sine", "3");
    add(&mut parameters, 3, "formula", "sin + sin(0) + sine");
    parameters.rename(1, "ratio").unwrap();
    assert_eq!(parameters.get(3).unwrap().source, "ratio + sin(0) + sine");
    assert_eq!(value(&parameters, 3), 5.0);
    parameters.set_expression(1, "4").unwrap();
    assert_eq!(value(&parameters, 3), 7.0);
}

#[test]
fn unit_names_and_unicode_positions_survive_multiple_renames() {
    let mut parameters = collection();
    add(&mut parameters, 1, "cm", "2 cm");
    add(&mut parameters, 2, "result", "cm + 5 cm + cm");
    for name in ["µwidth", "長さ", "w", "cm"] {
        parameters.rename(1, name).unwrap();
        assert_eq!(
            parameters.get(2).unwrap().source,
            format!("{name} + 5 cm + {name}")
        );
        assert_eq!(value(&parameters, 2), 9.0);
    }
    parameters.set_expression(1, "3 cm").unwrap();
    assert_eq!(value(&parameters, 2), 11.0);
}

#[test]
fn model_rename_flag_changes_only_for_an_actual_model_rename() {
    let mut parameters = collection();
    parameters
        .insert_expression(1, "d0", MODEL_PARAM, "2 cm")
        .unwrap();
    add(&mut parameters, 2, "user", "1");
    parameters.rename(1, "d0").unwrap();
    assert!(!parameters.get(1).unwrap().renamed);
    parameters.rename(1, "depth").unwrap();
    parameters.rename(2, "custom").unwrap();
    assert!(parameters.get(1).unwrap().renamed);
    assert!(!parameters.get(2).unwrap().renamed);
}

#[test]
fn malformed_or_duplicate_names_are_refused_before_any_mutation() {
    let mut parameters = collection();
    add(&mut parameters, 1, "a", "1");
    add(&mut parameters, 2, "b", "a + 1");
    parameters.drain_changed();
    let before = parameters.get(2).unwrap().clone();
    for name in ["b", "", " a", "a ", "1", "a+b", "sin(1)", "x y", "@", "a\n"] {
        let error = parameters.rename(1, name).unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert_eq!(parameters.get(2), Some(&before));
        assert_eq!(parameters.find("a"), Some(1));
        assert!(parameters.drain_changed().is_empty());
    }
    assert!(parameters.rename(999, "valid").is_err());
    assert!(
        parameters
            .insert_expression(3, "3bad", USER_PARAM, "1")
            .is_err()
    );
}

#[test]
fn new_name_resolves_pending_references_even_when_driver_ids_are_unchanged() {
    let mut parameters = collection();
    add(&mut parameters, 1, "width", "2 cm");
    add(&mut parameters, 2, "both", "width + span");
    add(&mut parameters, 3, "tail", "both * 2");
    assert!(parameters.get(2).unwrap().current_value().is_none());
    parameters.drain_changed();
    parameters.rename(1, "span").unwrap();
    assert_eq!(parameters.get(2).unwrap().source, "span + span");
    assert_eq!(value(&parameters, 2), 4.0);
    assert_eq!(value(&parameters, 3), 8.0);
    assert_eq!(parameters.drain_changed(), [2, 3]);
}

#[test]
fn a_new_name_can_resolve_a_cycle_without_using_stale_values() {
    let mut parameters = collection();
    add(&mut parameters, 1, "first", "future + 1");
    add(&mut parameters, 2, "second", "first + 1");
    parameters.rename(2, "future").unwrap();
    assert!(matches!(
        parameters.get(1).unwrap().health,
        ParameterHealth::Failed(ParameterFailure::Cycle(_))
    ));
    assert!(parameters.get(2).unwrap().current_value().is_none());
    parameters.set_expression(1, "5").unwrap();
    assert_eq!(value(&parameters, 2), 6.0);
}

#[test]
fn detached_failed_formulas_also_follow_bound_identity_renames() {
    let mut parameters = collection();
    add(&mut parameters, 1, "a", "2 cm");
    add(&mut parameters, 2, "b", "a + 1 cm");
    assert!(parameters.set_expression(1, "b + 1 cm").is_err());
    parameters.drain_changed();
    parameters.rename(2, "depth").unwrap();
    assert_eq!(parameters.get(1).unwrap().source, "depth + 1 cm");
    assert!(parameters.drivers(1).unwrap().is_empty());
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(1, "10 cm").unwrap();
    assert_eq!(value(&parameters, 2), 11.0);
}
