// SPDX-License-Identifier: GPL-2.0-only
//! Read the document-owned property table without changing the active model state.
use super::*;
use oblikovati_api::types::ModelStatePropertyKey;
use oblikovati_model::property::{DocumentProperties, SUMMARY_INFORMATION};
pub(super) fn available(
    members: &ModelStateMembers,
    properties: &DocumentProperties,
) -> Vec<ModelStatePropertyKey> {
    let mut keys: Vec<_> = ["Title", "Subject", "Author", "Category", "Comments"]
        .into_iter()
        .map(|name| ModelStatePropertyKey {
            set: SUMMARY_INFORMATION.into(),
            name: name.into(),
        })
        .collect();
    for set in properties.sets() {
        for property in set.properties() {
            let key = ModelStatePropertyKey {
                set: set.name.clone(),
                name: property.name.clone(),
            };
            if !keys.contains(&key) {
                keys.push(key);
            }
        }
    }
    for key in members
        .property_table()
        .bases()
        .iter()
        .map(|base| &base.property)
        .chain(
            members
                .property_table()
                .values()
                .iter()
                .map(|value| &value.property),
        )
    {
        if !keys.contains(key) {
            keys.push(key.clone());
        }
    }
    keys
}

/// Read the inherited value without changing the active member.
/// Example: `base(&members, &properties, &key)`.
pub(super) fn base(
    members: &ModelStateMembers,
    properties: &DocumentProperties,
    key: &ModelStatePropertyKey,
) -> Option<Box<oblikovati_api::types::Variant>> {
    members
        .property_table()
        .base(key)
        .map(|base| base.value.clone())
        .unwrap_or_else(|| properties.info(&key.set, &key.name).map(|info| info.value))
        .map(Box::new)
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::types::{Variant, VariantValue};
    use oblikovati_model::model_state::{ModelStatePropertyBase, ModelStatePropertyTable};

    #[test]
    fn retained_property_is_available_and_keeps_its_inherited_base() {
        let mut members = ModelStateMembers::default();
        let key = ModelStatePropertyKey {
            set: "Custom".into(),
            name: "Label".into(),
        };
        let value = Variant {
            value: VariantValue::String("Base".into()),
            unit: String::new(),
        };
        members
            .set_property_table(ModelStatePropertyTable::default().with_bases(vec![
                ModelStatePropertyBase {
                    property: key.clone(),
                    value: Some(value.clone()),
                },
            ]))
            .unwrap();
        let properties = DocumentProperties::default();
        assert!(available(&members, &properties).contains(&key));
        assert_eq!(base(&members, &properties, &key).as_deref(), Some(&value));
    }
}
