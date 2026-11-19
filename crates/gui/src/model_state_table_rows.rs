// SPDX-License-Identifier: GPL-2.0-only
//! Stage member row changes with draft identities until the host accepts them (#4119).
use oblikovati_api::wire::{ModelStateTableRow, ModelStateTableRowRef, ModelStateTableView};

pub(crate) fn insert(view: &mut ModelStateTableView, after: usize, sequence: &mut u64) -> bool {
    if !valid_cells(view) {
        return false;
    }
    let Some(rows) = view.rows.as_mut() else {
        return false;
    };
    let Some(source) = rows.get(after).cloned() else {
        return false;
    };
    let Some(key) = next_key(rows, sequence) else {
        return false;
    };
    let name = next_name(rows);
    let reference = ModelStateTableRowRef {
        draft_key: key,
        ..Default::default()
    };
    rows.insert(
        after + 1,
        ModelStateTableRow {
            ref_: reference.clone(),
            name,
            source: Some(Box::new(source.ref_)),
        },
    );
    for column in view.properties.as_mut().into_iter().flatten() {
        insert_cell(&mut column.cells, after, &reference);
    }
    for column in view.suppressions.as_mut().into_iter().flatten() {
        insert_cell(&mut column.cells, after, &reference);
    }
    for column in view.parameters.as_mut().into_iter().flatten() {
        insert_cell(&mut column.cells, after, &reference);
    }
    true
}
trait DraftCell: Clone {
    fn row_mut(&mut self) -> &mut ModelStateTableRowRef;
}
impl DraftCell for oblikovati_api::wire::ModelStateTablePropertyCell {
    fn row_mut(&mut self) -> &mut ModelStateTableRowRef {
        &mut self.row
    }
}
impl DraftCell for oblikovati_api::wire::ModelStateTableSuppressionCell {
    fn row_mut(&mut self) -> &mut ModelStateTableRowRef {
        &mut self.row
    }
}
impl DraftCell for oblikovati_api::wire::ModelStateTableParameterCell {
    fn row_mut(&mut self) -> &mut ModelStateTableRowRef {
        &mut self.row
    }
}
fn insert_cell<T: DraftCell>(
    cells: &mut Option<Vec<T>>,
    after: usize,
    reference: &ModelStateTableRowRef,
) {
    let Some(cells) = cells else {
        return;
    };
    let Some(mut cell) = cells.get(after).cloned() else {
        return;
    };
    *cell.row_mut() = reference.clone();
    cells.insert(after + 1, cell);
}
fn valid_cells(view: &ModelStateTableView) -> bool {
    let Some(rows) = view.rows.as_ref() else {
        return false;
    };
    let count = rows.len();
    view.properties
        .as_deref()
        .unwrap_or_default()
        .iter()
        .all(|column| {
            column
                .cells
                .as_ref()
                .is_some_and(|cells| cells.len() == count)
        })
        && view
            .suppressions
            .as_deref()
            .unwrap_or_default()
            .iter()
            .all(|column| {
                column
                    .cells
                    .as_ref()
                    .is_some_and(|cells| cells.len() == count)
            })
        && view
            .parameters
            .as_deref()
            .unwrap_or_default()
            .iter()
            .all(|column| {
                column
                    .cells
                    .as_ref()
                    .is_some_and(|cells| cells.len() == count)
            })
}
fn next_key(rows: &[ModelStateTableRow], sequence: &mut u64) -> Option<String> {
    loop {
        *sequence = sequence.checked_add(1)?;
        let key = format!("row-{sequence}");
        if !rows.iter().any(|row| row.ref_.draft_key == key) {
            return Some(key);
        }
    }
}
fn next_name(rows: &[ModelStateTableRow]) -> String {
    let mut index = 1usize;
    loop {
        let name = format!("Model State{index}");
        if !rows.iter().any(|row| row.name == name) {
            return name;
        }
        index += 1;
    }
}
pub(crate) fn can_delete(view: &ModelStateTableView, index: usize) -> bool {
    let Some(rows) = view.rows.as_deref() else {
        return false;
    };
    let Some(row) = rows.get(index) else {
        return false;
    };
    index != 0
        && !rows
            .iter()
            .any(|other| other.source.as_deref() == Some(&row.ref_))
}
pub(crate) fn delete(view: &mut ModelStateTableView, index: usize) -> bool {
    if !valid_cells(view) || !can_delete(view, index) {
        return false;
    }
    if let Some(rows) = view.rows.as_mut() {
        rows.remove(index);
    }
    for column in view.properties.as_mut().into_iter().flatten() {
        remove_cell(&mut column.cells, index);
    }
    for column in view.suppressions.as_mut().into_iter().flatten() {
        remove_cell(&mut column.cells, index);
    }
    for column in view.parameters.as_mut().into_iter().flatten() {
        remove_cell(&mut column.cells, index);
    }
    true
}

fn remove_cell<T>(cells: &mut Option<Vec<T>>, index: usize) {
    if let Some(cells) = cells {
        cells.remove(index);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::wire::{ModelStateTableParameter, ModelStateTableParameterCell};
    #[test]
    fn inserted_row_uses_a_draft_key_and_copies_parameter_overrides() {
        let primary = ModelStateTableRowRef {
            member_id: 1,
            ..Default::default()
        };
        let other = ModelStateTableRowRef {
            member_id: 2,
            ..Default::default()
        };
        let mut view = ModelStateTableView {
            rows: Some(vec![
                ModelStateTableRow {
                    ref_: primary.clone(),
                    name: "Primary".into(),
                    source: None,
                },
                ModelStateTableRow {
                    ref_: other.clone(),
                    name: "Copy".into(),
                    source: None,
                },
            ]),
            parameters: Some(vec![ModelStateTableParameter {
                parameter_id: 3,
                base: "1 cm".into(),
                cells: Some(vec![
                    ModelStateTableParameterCell {
                        row: primary,
                        expression: None,
                    },
                    ModelStateTableParameterCell {
                        row: other,
                        expression: Some(Box::new("2 cm".into())),
                    },
                ]),
            }]),
            ..Default::default()
        };
        assert!(insert(&mut view, 1, &mut 0));
        let rows = view.rows.as_ref().unwrap();
        assert_eq!(rows[2].name, "Model State1");
        assert_eq!(rows[2].ref_.member_id, 0);
        assert!(!rows[2].ref_.draft_key.is_empty());
        let cell = &view.parameters.as_ref().unwrap()[0].cells.as_ref().unwrap()[2];
        assert_eq!(cell.row, rows[2].ref_);
        assert_eq!(cell.expression.as_deref().map(String::as_str), Some("2 cm"));
        assert!(!can_delete(&view, 0));
        assert!(delete(&mut view, 2));
        assert_eq!(view.rows.as_ref().unwrap().len(), 2);
    }
}
