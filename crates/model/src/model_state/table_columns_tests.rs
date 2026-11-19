// SPDX-License-Identifier: GPL-2.0-only
//! Mixed table columns retain one visible order across categories.
use super::*;
use oblikovati_api::types::ModelStatePropertyKey;

#[test]
fn mixed_columns_keep_order_and_hidden_data() {
    let mut members = ModelStateMembers::default();
    let key = ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    };
    members
        .set_property_table(ModelStatePropertyTable::new(vec![key.clone()], vec![]))
        .unwrap();
    members
        .set_parameter_table(ModelStateParameterTable::new(
            vec![ModelStateParameterBase {
                parameter: 9,
                expression: "1 mm".into(),
            }],
            vec![],
        ))
        .unwrap();
    let order = vec![
        ModelStateTableColumn::Parameter(9),
        ModelStateTableColumn::Property(key.clone()),
        ModelStateTableColumn::Suppression(7),
    ];
    members.set_table_columns(order.clone()).unwrap();
    let encoded = serde_json::to_vec(&members).unwrap();
    let restored: ModelStateMembers = serde_json::from_slice(&encoded).unwrap();
    restored.validate().unwrap();
    assert_eq!(restored.table_columns(), order);
    members.set_table_columns(vec![]).unwrap();
    assert!(members.property_table().base(&key).is_some());
    assert_eq!(members.parameter_table().expression(1, 9), Some("1 mm"));
}
#[test]
fn duplicate_or_unowned_columns_are_refused() {
    let mut members = ModelStateMembers::default();
    assert!(
        members
            .set_table_columns(vec![ModelStateTableColumn::Parameter(9)])
            .is_err()
    );
    assert!(
        members
            .set_table_columns(vec![
                ModelStateTableColumn::Suppression(7),
                ModelStateTableColumn::Suppression(7)
            ])
            .is_err()
    );
    assert!(members.table_columns().is_empty());
}
