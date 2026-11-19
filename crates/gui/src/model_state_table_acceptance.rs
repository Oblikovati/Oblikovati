// SPDX-License-Identifier: GPL-2.0-only
//! Compare an accepted table after the host replaces draft row keys (#4119).
use oblikovati_api::wire::*;

pub(crate) fn same_table(draft: &ModelStateTableView, current: &ModelStateTableView) -> bool {
    rows_match(draft, current)
        && draft.columns == current.columns
        && properties_match(draft, current)
        && suppressions_match(draft, current)
        && parameters_match(draft, current)
}
fn rows_match(draft: &ModelStateTableView, current: &ModelStateTableView) -> bool {
    let (Some(left), Some(right)) = (&draft.rows, &current.rows) else {
        return false;
    };
    left.len() == right.len()
        && left.iter().zip(right).all(|(a, b)| {
            a.name == b.name
                && b.ref_.member_id != 0
                && (a.ref_.member_id == 0 || a.ref_.member_id == b.ref_.member_id)
        })
}
fn row_matches(
    draft: &ModelStateTableView,
    current: &ModelStateTableView,
    index: usize,
    a: &ModelStateTableRowRef,
    b: &ModelStateTableRowRef,
) -> bool {
    draft
        .rows
        .as_ref()
        .and_then(|rows| rows.get(index))
        .is_some_and(|row| row.ref_ == *a)
        && current
            .rows
            .as_ref()
            .and_then(|rows| rows.get(index))
            .is_some_and(|row| row.ref_ == *b)
}
fn properties_match(draft: &ModelStateTableView, current: &ModelStateTableView) -> bool {
    let (Some(left), Some(right)) = (&draft.properties, &current.properties) else {
        return false;
    };
    left.len() == right.len()
        && left.iter().all(|a| {
            right
                .iter()
                .find(|b| b.property == a.property)
                .is_some_and(|b| {
                    a.base == b.base
                        && match (&a.cells, &b.cells) {
                            (Some(x), Some(y)) => {
                                x.len() == y.len()
                                    && x.iter().zip(y).enumerate().all(|(i, (x, y))| {
                                        x.value == y.value
                                            && row_matches(draft, current, i, &x.row, &y.row)
                                    })
                            }
                            _ => false,
                        }
                })
        })
}
fn suppressions_match(draft: &ModelStateTableView, current: &ModelStateTableView) -> bool {
    let (Some(left), Some(right)) = (&draft.suppressions, &current.suppressions) else {
        return false;
    };
    left.len() == right.len()
        && left.iter().all(|a| {
            right
                .iter()
                .find(|b| b.feature_id == a.feature_id)
                .is_some_and(|b| {
                    a.base == b.base
                        && match (&a.cells, &b.cells) {
                            (Some(x), Some(y)) => {
                                x.len() == y.len()
                                    && x.iter().zip(y).enumerate().all(|(i, (x, y))| {
                                        x.suppressed == y.suppressed
                                            && row_matches(draft, current, i, &x.row, &y.row)
                                    })
                            }
                            _ => false,
                        }
                })
        })
}
fn parameters_match(draft: &ModelStateTableView, current: &ModelStateTableView) -> bool {
    let (Some(left), Some(right)) = (&draft.parameters, &current.parameters) else {
        return false;
    };
    left.len() == right.len()
        && left.iter().all(|a| {
            right
                .iter()
                .find(|b| b.parameter_id == a.parameter_id)
                .is_some_and(|b| {
                    a.base == b.base
                        && match (&a.cells, &b.cells) {
                            (Some(x), Some(y)) => {
                                x.len() == y.len()
                                    && x.iter().zip(y).enumerate().all(|(i, (x, y))| {
                                        x.expression == y.expression
                                            && row_matches(draft, current, i, &x.row, &y.row)
                                    })
                            }
                            _ => false,
                        }
                })
        })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn accepted_new_row_may_receive_a_persistent_identity() {
        let draft_row = ModelStateTableRowRef {
            draft_key: "row-1".into(),
            ..Default::default()
        };
        let mut draft = ModelStateTableView {
            rows: Some(vec![ModelStateTableRow {
                ref_: draft_row.clone(),
                name: "Model State1".into(),
                source: None,
            }]),
            columns: Some(vec![]),
            properties: Some(vec![]),
            suppressions: Some(vec![]),
            parameters: Some(vec![ModelStateTableParameter {
                parameter_id: 7,
                base: "1 cm".into(),
                cells: Some(vec![ModelStateTableParameterCell {
                    row: draft_row,
                    expression: Some(Box::new("2 cm".into())),
                }]),
            }]),
            ..Default::default()
        };
        let mut accepted = draft.clone();
        let id = ModelStateTableRowRef {
            member_id: 9,
            ..Default::default()
        };
        accepted.rows.as_mut().unwrap()[0].ref_ = id.clone();
        accepted.parameters.as_mut().unwrap()[0]
            .cells
            .as_mut()
            .unwrap()[0]
            .row = id;
        assert!(same_table(&draft, &accepted));
        draft.parameters.as_mut().unwrap()[0]
            .cells
            .as_mut()
            .unwrap()[0]
            .expression = Some(Box::new("3 cm".into()));
        assert!(!same_table(&draft, &accepted));
    }
}
