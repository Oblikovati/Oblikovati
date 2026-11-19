// SPDX-License-Identifier: GPL-2.0-only
//! Model-state property values retain their owners and survive column visibility changes.
use super::*;
use oblikovati_api::types::{ModelStatePropertyKey, Variant, VariantValue};

fn category() -> ModelStatePropertyKey {
    ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    }
}
fn value(member: u64, text: &str) -> ModelStatePropertyValue {
    ModelStatePropertyValue {
        member,
        property: category(),
        value: Variant {
            value: VariantValue::String(text.into()),
            unit: String::new(),
        },
    }
}

#[test]
fn removing_and_restoring_a_column_preserves_member_values() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Prototype").unwrap();
    let original = ModelStatePropertyTable::new(vec![category()], vec![value(2, "Review")]);
    members.set_property_table(original.clone()).unwrap();
    members
        .set_table_columns(vec![ModelStateTableColumn::Property(category())])
        .unwrap();
    members.set_table_columns(vec![]).unwrap();
    let hidden = ModelStatePropertyTable::new(vec![], original.values().to_vec());
    members.set_property_table(hidden).unwrap();
    assert!(members.table_columns().is_empty());
    assert_eq!(
        members.property_table().value(2, &category()),
        Some(&value(2, "Review").value)
    );
    members.set_property_table(original.clone()).unwrap();
    assert_eq!(members.property_table(), &original);
}

#[test]
fn invalid_tables_are_rejected_without_replacing_the_previous_table() {
    let mut members = ModelStateMembers::default();
    let original = ModelStatePropertyTable::new(vec![category()], vec![value(1, "First")]);
    members.set_property_table(original.clone()).unwrap();
    let duplicate = vec![
        ModelStateTableColumn::Property(category()),
        ModelStateTableColumn::Property(category()),
    ];
    assert!(members.set_table_columns(duplicate).is_err());
    let foreign = ModelStatePropertyTable::new(vec![category()], vec![value(88, "Other")]);
    assert!(members.set_property_table(foreign).is_err());
    let repeated =
        ModelStatePropertyTable::new(vec![category()], vec![value(1, "A"), value(1, "B")]);
    assert!(members.set_property_table(repeated).is_err());
    let mut unit_value = value(1, "A");
    unit_value.value.unit = "mm".into();
    let unit_table = ModelStatePropertyTable::new(vec![category()], vec![unit_value]);
    assert!(members.set_property_table(unit_table).is_err());
    assert_eq!(members.property_table(), &original);
}

#[test]
fn a_property_without_an_inherited_base_is_rejected() {
    let mut members = ModelStateMembers::default();
    let table =
        ModelStatePropertyTable::new(vec![category()], vec![value(1, "Review")]).with_bases(vec![]);
    assert!(members.set_property_table(table).is_err());
    assert!(members.table_columns().is_empty());
}

#[test]
fn copying_and_deleting_members_preserves_other_values() {
    let mut members = ModelStateMembers::default();
    members
        .set_property_table(ModelStatePropertyTable::new(
            vec![category()],
            vec![value(1, "Primary")],
        ))
        .unwrap();
    members.insert(2, "Prototype").unwrap();
    assert_eq!(
        members.property_table().value(2, &category()),
        Some(&value(2, "Primary").value)
    );
    members.copy(2, 3, "Copy").unwrap();
    members.delete(2).unwrap();
    assert_eq!(members.property_table().value(2, &category()), None);
    assert_eq!(
        members.property_table().value(3, &category()),
        Some(&value(3, "Primary").value)
    );
    assert!(members.validate().is_ok());
}

#[test]
fn unset_member_value_inherits_the_retained_base() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Prototype").unwrap();
    let base = ModelStatePropertyBase {
        property: category(),
        value: Some(value(1, "Design").value),
    };
    let table = ModelStatePropertyTable::new(vec![category()], vec![value(2, "Review")])
        .with_bases(vec![base]);
    members.set_property_table(table).unwrap();
    assert_eq!(
        members.property_table().effective_value(1, &category()),
        Some(&value(1, "Design").value)
    );
    assert_eq!(
        members.property_table().effective_value(2, &category()),
        Some(&value(2, "Review").value)
    );
}

#[test]
fn property_table_survives_serialization_and_preserves_numeric_bits() {
    let mut members = ModelStateMembers::default();
    let mut entry = value(1, "ignored");
    entry.value.value = VariantValue::Double(-0.0);
    members
        .set_property_table(ModelStatePropertyTable::new(vec![category()], vec![entry]))
        .unwrap();
    let json = serde_json::to_vec(&members).unwrap();
    let restored: ModelStateMembers = serde_json::from_slice(&json).unwrap();
    restored.validate().unwrap();
    assert_eq!(restored, members);
    let VariantValue::Double(number) = &restored
        .property_table()
        .value(1, &category())
        .unwrap()
        .value
    else {
        panic!("expected a numeric property")
    };
    assert_eq!(number.to_bits(), (-0.0_f64).to_bits());
}
