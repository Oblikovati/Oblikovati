// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{parameter::ParameterHealth, units::UnitsOfMeasure};
use oblikovati_persistence::package::DocumentPackage;

fn package(parameters: &str) -> DocumentPackage {
    let source =
        format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters:\n{parameters}\n");
    DocumentPackage::decode(source.as_bytes()).unwrap()
}

#[test]
fn restored_forward_references_form_live_editable_dependencies() {
    let package = package(
        "    - {name: twice, kind: user, expression: 'width * 2'}\n    - {name: width, kind: model, expression: '5 mm'}",
    );
    let mut restored = package
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(restored.parameters.ids(), [1, 2]);
    assert_eq!(restored.parameters.drivers(1).unwrap(), [2]);
    assert_eq!(
        restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        1.0
    );
    assert!(restored.parameters.drain_changed().is_empty());
    restored.parameters.set_expression(2, "10 mm").unwrap();
    assert_eq!(
        restored
            .parameters
            .get(1)
            .unwrap()
            .current_value()
            .unwrap()
            .value,
        2.0
    );
    assert_eq!(restored.ids.allocate().unwrap(), 3);
}

#[test]
fn numeric_kinds_and_literal_flavors_are_restored() {
    let source = "    - {name: user, kind: user, expression: '1'}\n    - {name: model, kind: model, expression: '2'}\n    - {name: table, kind: table, expression: '3'}\n    - {name: measured, kind: reference, value: 4, unit: length}\n    - {name: linked, kind: derived, value: 5, unit: angle}\n    - {name: label, kind: user, valueType: text, text: 'Ω'}\n    - {name: enabled, kind: user, valueType: boolean, bool: true}";
    let restored = package(source)
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    for (id, kind) in [
        (1, USER_PARAM),
        (2, MODEL_PARAM),
        (3, TABLE_PARAM),
        (4, REFERENCE_PARAM),
        (5, DERIVED_PARAM),
    ] {
        assert_eq!(restored.parameters.get(id).unwrap().kind, kind);
        assert_eq!(
            restored
                .parameters
                .get(id)
                .unwrap()
                .current_value()
                .unwrap()
                .value,
            id as f64
        );
    }
    assert_eq!(restored.parameters.text_value(6).unwrap(), "Ω");
    assert!(restored.parameters.boolean_value(7).unwrap());
}

#[test]
fn annotations_choices_and_introspection_survive_restoration() {
    let source = "    - {name: width, kind: model, expression: '2', comment: note, key: true, export: true, hidden: true, precision: 4, displayFormat: fractional, renamed: true, disabledActions: [edit, rename, delete], expressionList: ['3', '1', '2'], allowCustomValue: true, sortedValueList: true}";
    let restored = package(source)
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    let state = restored.parameters.get(1).unwrap();
    assert_eq!(state.annotations.comment, "note");
    assert!(state.annotations.is_key && state.annotations.exposed_as_property);
    assert!(!state.annotations.visible);
    assert_eq!(state.annotations.precision, 4);
    assert_eq!(state.annotations.display_format, DISPLAY_FORMAT_FRACTIONAL);
    assert!(state.renamed);
    assert_eq!(state.disabled_actions, ActionType(7));
    assert_eq!(
        state.choices.expressions.as_ref().unwrap(),
        &["1", "2", "3"]
    );
    assert!(state.choices.allow_custom_values && !state.choices.custom_order);
}

#[test]
fn malformed_unsupported_or_unhealthy_recipes_are_refused() {
    let records = [
        "{name: a, kind: unknown}",
        "{name: a, kind: user, valueType: unknown}",
        "{name: a, kind: reference, value: 1, unit: unknown}",
        "{name: a, kind: reference, value: .inf, unit: length}",
        "{name: a, kind: user, expression: 'missing + 1'}",
        "{name: a, kind: user, expression: '1 / 0'}",
        "{name: a, kind: user, expression: '1', tolerance: {type: symmetric, upper: 1}}",
        "{name: a, kind: user, expression: '1', disabledActions: [unknown]}",
        "{name: a, kind: user, expression: '1', displayFormat: unknown}",
    ];
    for record in records {
        assert!(
            package(&format!("    - {record}"))
                .restore_parameters(UnitsOfMeasure::default())
                .is_err(),
            "{record}"
        );
    }
}

#[test]
fn duplicates_and_cycles_cannot_replace_an_existing_live_table() {
    let original = package("    - {name: original, kind: user, expression: '4'}");
    let restored = original
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    for source in [
        "    - {name: a, kind: user, expression: 'b'}\n    - {name: b, kind: user, expression: 'a'}",
        "    - {name: a, kind: user, expression: '1'}\n    - {name: a, kind: user, expression: '2'}",
    ] {
        assert!(
            package(source)
                .restore_parameters(UnitsOfMeasure::default())
                .is_err()
        );
        assert_eq!(restored.parameters.find("original"), Some(1));
        assert_eq!(
            restored.parameters.get(1).unwrap().health,
            ParameterHealth::Healthy
        );
    }
}

#[test]
fn absent_parameters_are_empty_but_unimplemented_related_state_is_not_dropped() {
    for suffix in ["", "model: null", "model: {}", "model: {parameters: null}"] {
        let input = format!("schemaVersion: 2\ndocumentType: 1\n{suffix}");
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        assert!(
            package
                .restore_parameters(UnitsOfMeasure::default())
                .unwrap()
                .parameters
                .ids()
                .is_empty()
        );
    }
    for name in ["parameterSettings", "derivedParameterTables"] {
        let input = format!("schemaVersion: 2\ndocumentType: 1\nmodel: {{{name}: {{}}}}");
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        assert!(
            package
                .restore_parameters(UnitsOfMeasure::default())
                .unwrap_err()
                .to_string()
                .contains(name)
        );
    }
}

#[test]
fn authored_choice_order_and_architectural_format_are_preserved() {
    let source = "    - {name: label, kind: user, valueType: text, text: abc, expressionList: [z, a], displayFormat: architectural}";
    let restored = package(source)
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    let state = restored.parameters.get(1).unwrap();
    assert_eq!(
        state.annotations.display_format,
        DISPLAY_FORMAT_ARCHITECTURAL
    );
    assert_eq!(state.choices.expressions.as_ref().unwrap(), &["z", "a"]);
    assert!(state.choices.custom_order);
}

#[test]
fn conflicting_value_fields_and_orphaned_choice_flags_are_refused() {
    for record in [
        "{name: a, kind: user, expression: '1', text: hidden}",
        "{name: a, kind: user, valueType: text, text: visible, bool: true}",
        "{name: a, kind: user, valueType: boolean, text: hidden}",
        "{name: a, kind: reference, value: 1, unit: length, expression: hidden}",
        "{name: a, kind: user, expression: '1', allowCustomValue: true}",
    ] {
        assert!(
            package(&format!("    - {record}"))
                .restore_parameters(UnitsOfMeasure::default())
                .is_err()
        );
    }
}
