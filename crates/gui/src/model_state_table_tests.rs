// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    types::Variant,
    wire::{ModelStateTablePropertyChoice, ModelStateTableRow},
};
fn state() -> GuiCommandState {
    let key = ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    };
    GuiCommandState {
        document: Some(1),
        model_state_table: Some(std::sync::Arc::new(ModelStateTableView {
            generation: 4,
            rows: Some(vec![ModelStateTableRow {
                ref_: ModelStateTableRowRef {
                    member_id: 1,
                    ..Default::default()
                },
                name: "Primary".into(),
                source: None,
            }]),
            available_properties: Some(vec![ModelStateTablePropertyChoice {
                property: key,
                base: None,
            }]),
            columns: Some(vec![]),
            properties: Some(vec![]),
            suppressions: Some(vec![]),
            parameters: Some(vec![]),
            ..Default::default()
        })),
        ..Default::default()
    }
}
fn choose_property(table: &mut ModelStateTable) {
    table.available = table
        .draft
        .as_ref()
        .unwrap()
        .available_properties
        .as_ref()
        .unwrap()
        .first()
        .map(|choice| choice.property.clone());
}
#[test]
fn cell_values_preserve_type_and_reject_invalid_numbers() {
    let integer = Variant {
        value: VariantValue::Integer(1),
        unit: String::new(),
    };
    let number = Variant {
        value: VariantValue::Double(1.0),
        unit: String::new(),
    };
    assert!(matches!(
        parse_cell(Some(&integer), "42").unwrap().value,
        VariantValue::Integer(42)
    ));
    assert!(parse_cell(Some(&integer), "4.2").is_none());
    assert!(parse_cell(Some(&number), "NaN").is_none());
    assert!(
        matches!(parse_cell(None, "Review").unwrap().value, VariantValue::String(text) if text == "Review")
    );
}
#[test]
fn cancel_discards_a_staged_column() {
    let state = state();
    let mut table = ModelStateTable::default();
    table.open(&state);
    choose_property(&mut table);
    table.add();
    assert!(table.dirty);
    table.close();
    table.open(&state);
    assert!(
        table
            .draft
            .as_ref()
            .unwrap()
            .columns
            .as_ref()
            .unwrap()
            .is_empty()
    );
    assert!(!table.dirty);
}
#[test]
fn removing_and_readding_keeps_authored_values() {
    let state = state();
    let mut table = ModelStateTable::default();
    table.open(&state);
    choose_property(&mut table);
    let key = table.available.clone().unwrap();
    table.add();
    table.draft.as_mut().unwrap().properties.as_mut().unwrap()[0]
        .cells
        .as_mut()
        .unwrap()[0]
        .value = Some(Box::new(Variant {
        value: VariantValue::String("Review".into()),
        unit: String::new(),
    }));
    table.chosen = Some(key);
    table.remove();
    table.add();
    let value = &table.draft.as_ref().unwrap().properties.as_ref().unwrap()[0]
        .cells
        .as_ref()
        .unwrap()[0]
        .value;
    assert!(
        matches!(&value.as_ref().unwrap().value, VariantValue::String(value) if value == "Review")
    );
}
#[test]
fn concurrent_document_change_keeps_the_draft_for_cancel() {
    let mut state = state();
    let mut table = ModelStateTable::default();
    table.open(&state);
    table.pending = true;
    let view = std::sync::Arc::make_mut(state.model_state_table.as_mut().unwrap());
    view.generation += 1;
    view.columns = Some(vec![ModelStateTableColumnRef::default()]);
    table.sync_before_docking(&state);
    assert!(table.document.is_some());
    assert!(table.stale(&state));
    assert!(!table.pending);
}
#[test]
fn accepted_draft_closes_only_after_matching_document_result() {
    let mut state = state();
    let mut table = ModelStateTable::default();
    table.open(&state);
    choose_property(&mut table);
    table.add();
    table.pending = true;
    let current = std::sync::Arc::make_mut(state.model_state_table.as_mut().unwrap());
    *current = table.draft.as_ref().unwrap().clone();
    current.generation += 1;
    table.sync_before_docking(&state);
    assert!(table.document.is_none());
}
#[test]
fn property_edit_preserves_other_unified_columns() {
    let mut state = state();
    let view = std::sync::Arc::make_mut(state.model_state_table.as_mut().unwrap());
    view.columns = Some(vec![ModelStateTableColumnRef {
        kind: oblikovati_api::types::MODEL_STATE_TABLE_PARAMETER_COLUMN,
        parameter_id: 7,
        ..Default::default()
    }]);
    view.parameters = Some(vec![oblikovati_api::wire::ModelStateTableParameter {
        parameter_id: 7,
        base: "10 mm".into(),
        cells: Some(vec![]),
    }]);
    let mut table = ModelStateTable::default();
    table.open(&state);
    choose_property(&mut table);
    table.add();
    let request = request(table.draft.as_ref().unwrap());
    assert_eq!(request.parameters.as_ref().unwrap()[0].base, "10 mm");
    assert_eq!(request.columns.as_ref().unwrap()[0].parameter_id, 7);
    assert_eq!(request.properties.as_ref().unwrap().len(), 1);
}

#[test]
fn verification_is_visible_only_for_the_exact_document_and_draft() {
    let mut state = state();
    let mut table = ModelStateTable::default();
    table.open(&state);
    let draft = request(table.draft.as_ref().unwrap());
    state.model_state_verification = Some((
        1,
        draft,
        oblikovati_api::wire::ModelStateTableVerification {
            valid: true,
            diagnostics: Some(vec![]),
        },
    ));
    assert!(table.current_verification(&state).unwrap().valid);
    choose_property(&mut table);
    table.add();
    assert!(table.current_verification(&state).is_none());
    table.document = Some(2);
    assert!(table.current_verification(&state).is_none());
}
