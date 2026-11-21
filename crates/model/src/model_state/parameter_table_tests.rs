// SPDX-License-Identifier: GPL-2.0-only
//! Parameter expressions retain member identity and inherited values.
use super::*;

fn table() -> ModelStateParameterTable {
    ModelStateParameterTable::new(
        vec![ModelStateParameterBase {
            parameter: 4,
            expression: "1 in".into(),
        }],
        vec![ModelStateParameterValue {
            member: 2,
            parameter: 4,
            expression: "2 in".into(),
        }],
    )
}
#[test]
fn member_expressions_inherit_and_survive_copy_and_delete() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Second").unwrap();
    members.set_parameter_table(table()).unwrap();
    assert_eq!(members.parameter_table().expression(1, 4), Some("1 in"));
    assert_eq!(members.parameter_table().expression(2, 4), Some("2 in"));
    members.copy(2, 3, "Copy").unwrap();
    members.delete(2).unwrap();
    assert_eq!(members.parameter_table().expression(3, 4), Some("2 in"));
    assert!(
        !members
            .parameter_table()
            .values()
            .iter()
            .any(|value| value.member == 2)
    );
}
#[test]
fn invalid_parameter_table_does_not_replace_existing_values() {
    let mut members = ModelStateMembers::default();
    assert!(
        members
            .set_table_columns(vec![ModelStateTableColumn::Parameter(4)])
            .is_err()
    );
    let invalid = ModelStateParameterTable::new(
        vec![ModelStateParameterBase {
            parameter: 4,
            expression: "1 in".into(),
        }],
        vec![ModelStateParameterValue {
            member: 99,
            parameter: 4,
            expression: "2 in".into(),
        }],
    );
    assert!(members.set_parameter_table(invalid).is_err());
    assert!(members.table_columns().is_empty());
}
#[test]
fn native_recipe_round_trip_preserves_expressions_and_hidden_values() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Second").unwrap();
    members.set_parameter_table(table()).unwrap();
    let hidden = ModelStateParameterTable::new(
        members.parameter_table().bases().to_vec(),
        members.parameter_table().values().to_vec(),
    );
    members.set_parameter_table(hidden).unwrap();
    let encoded = serde_json::to_vec(&members).unwrap();
    let restored: ModelStateMembers = serde_json::from_slice(&encoded).unwrap();
    restored.validate().unwrap();
    assert_eq!(restored.parameter_table().expression(2, 4), Some("2 in"));
    assert!(restored.table_columns().is_empty());
}

#[test]
fn reordered_members_keep_identity_and_expressions() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Second").unwrap();
    members.insert(3, "Third").unwrap();
    members.set_parameter_table(table()).unwrap();
    members.reorder(&[1, 3, 2]).unwrap();
    assert_eq!(
        members
            .list()
            .iter()
            .map(|member| member.id)
            .collect::<Vec<_>>(),
        vec![1, 3, 2]
    );
    assert_eq!(members.parameter_table().expression(2, 4), Some("2 in"));
    assert!(members.reorder(&[1, 3, 3]).is_err());
    assert_eq!(
        members
            .list()
            .iter()
            .map(|member| member.id)
            .collect::<Vec<_>>(),
        vec![1, 3, 2]
    );
}
