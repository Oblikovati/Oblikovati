// SPDX-License-Identifier: GPL-2.0-only
//! Shared model-state table editing helpers (#4119).
use super::*;

pub(super) fn property_visible(view: &ModelStateTableView, key: &ModelStatePropertyKey) -> bool {
    view.columns
        .as_deref()
        .unwrap_or_default()
        .iter()
        .any(|column| {
            column.kind == MODEL_STATE_TABLE_PROPERTY_COLUMN
                && column.property.as_deref() == Some(key)
        })
}
pub(super) fn new_column(
    view: &ModelStateTableView,
    key: ModelStatePropertyKey,
) -> ModelStateTableProperty {
    let base = view
        .available_properties
        .as_deref()
        .unwrap_or_default()
        .iter()
        .find(|choice| choice.property == key)
        .and_then(|choice| choice.base.clone());
    let cells = view
        .rows
        .as_deref()
        .unwrap_or_default()
        .iter()
        .map(|row| ModelStateTablePropertyCell {
            row: row.ref_.clone(),
            value: None,
        })
        .collect();
    ModelStateTableProperty {
        property: key,
        base,
        cells: Some(cells),
    }
}
pub(super) fn row_key(row: &ModelStateTableRowRef) -> String {
    if row.member_id != 0 {
        format!("member-{}", row.member_id)
    } else {
        format!("draft-{}", row.draft_key)
    }
}
pub(super) fn edit_cell(
    ui: &Ui,
    row: &ModelStateTableRowRef,
    index: usize,
    column: &mut ModelStateTableProperty,
    buffers: &mut BTreeMap<CellKey, String>,
    invalid: &mut BTreeSet<CellKey>,
) -> bool {
    let Some(cell) = column.cells.as_mut().and_then(|cells| cells.get_mut(index)) else {
        return false;
    };
    let value = cell.value.as_deref().or(column.base.as_deref());
    let key = (
        row_key(row),
        column.property.set.clone(),
        column.property.name.clone(),
    );
    let label = format!("##state-cell-{}-{}-{}", key.0, key.1, key.2);
    let text = buffers
        .entry(key.clone())
        .or_insert_with(|| value.map_or_else(String::new, property_text));
    let disabled = ui.begin_disabled(value.is_some_and(|value| {
        matches!(
            value.value,
            VariantValue::Bytes(_) | VariantValue::Uninitialized
        )
    }));
    let changed = ui.input_text(&label, text).build();
    drop(disabled);
    if !changed {
        return false;
    }
    match parse_cell(value, text) {
        Some(value) => {
            cell.value = Some(Box::new(value));
            invalid.remove(&key);
        }
        None => {
            invalid.insert(key);
        }
    }
    true
}
pub(super) fn request(view: &ModelStateTableView) -> SetModelStateTableArgs {
    SetModelStateTableArgs {
        expected_generation: view.generation,
        rows: view.rows.clone(),
        columns: view.columns.clone(),
        properties: view.properties.clone(),
        suppressions: view.suppressions.clone(),
        parameters: view.parameters.clone(),
    }
}
