// SPDX-License-Identifier: GPL-2.0-only
//! Check property ownership and inherited values for table edits.
use super::*;
use oblikovati_model::model_state::ModelStatePropertyBase;

pub(super) fn same_default(
    first: &Option<Box<oblikovati_api::types::Variant>>,
    second: &Option<Box<oblikovati_api::types::Variant>>,
) -> bool {
    let property = oblikovati_api::types::ModelStatePropertyKey::default();
    ModelStatePropertyBase {
        property: property.clone(),
        value: first.as_deref().cloned(),
    } == ModelStatePropertyBase {
        property,
        value: second.as_deref().cloned(),
    }
}

pub(super) fn validate_ownership(
    snapshot: &DocumentSnapshot,
    key: &oblikovati_api::types::ModelStatePropertyKey,
) -> Result<(), Error> {
    if snapshot
        .properties
        .get(&key.set, &key.name)
        .is_some_and(|property| !property.from_parameter.is_empty())
        || crate::properties::exposed_parameter(&snapshot.parameters, &key.set, &key.name).is_some()
    {
        return Err(Error(format!(
            "property {key:?}: expected a property not owned by a parameter"
        )));
    }
    Ok(())
}
