// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{ModelStatePropertyKey, Variant, VariantValue};
use oblikovati_model::model_state::{
    ModelStatePropertyBase, ModelStatePropertyTable, ModelStatePropertyValue,
};

#[test]
fn switching_member_projects_override_and_inherited_absence() {
    let mut members = ModelStateMembers::default();
    members.insert(2, "Prototype").unwrap();
    let key = ModelStatePropertyKey {
        set: "Summary Information".into(),
        name: "Category".into(),
    };
    let value = Variant {
        value: VariantValue::String("Review".into()),
        unit: String::new(),
    };
    let table = ModelStatePropertyTable::new(
        vec![key.clone()],
        vec![ModelStatePropertyValue {
            member: 2,
            property: key.clone(),
            value: value.clone(),
        }],
    )
    .with_bases(vec![ModelStatePropertyBase {
        property: key.clone(),
        value: None,
    }]);
    members.set_property_table(table).unwrap();
    let mut properties = DocumentProperties::default();
    members.activate(2).unwrap();
    project_values(&members, &mut properties).unwrap();
    assert_eq!(
        properties.get(&key.set, &key.name).unwrap().value,
        value.value
    );
    members.activate(1).unwrap();
    project_values(&members, &mut properties).unwrap();
    assert!(properties.get(&key.set, &key.name).is_none());
}
