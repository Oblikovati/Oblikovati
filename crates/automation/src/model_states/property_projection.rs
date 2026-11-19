// SPDX-License-Identifier: GPL-2.0-only
//! Project a model state's retained values into document properties.
use super::*;
use oblikovati_model::property::DocumentProperties;

pub(super) fn project(snapshot: &mut DocumentSnapshot) -> Result<(), Error> {
    let Some(members) = snapshot.model_states.as_ref() else {
        return Ok(());
    };
    project_values(members, &mut snapshot.properties)
}

fn project_values(
    members: &ModelStateMembers,
    properties: &mut DocumentProperties,
) -> Result<(), Error> {
    let active = members
        .active()
        .ok_or_else(|| Error("model-state projection: expected an active member".into()))?
        .id;
    for base in members.property_table().bases() {
        let key = &base.property;
        match members.property_table().effective_value(active, key) {
            Some(value) => properties
                .put(&key.set, &key.name, value.value.clone())
                .map_err(|cause| Error(cause.to_string()))?,
            None => {
                properties.remove(&key.set, &key.name);
            }
        }
    }
    Ok(())
}

pub(crate) fn record_edit(
    snapshot: &mut DocumentSnapshot,
    property: &PropertyInfo,
) -> Result<(), Error> {
    use oblikovati_api::types::ModelStatePropertyKey;
    use oblikovati_model::model_state::{
        ModelStateEditScope, ModelStatePropertyTable, ModelStatePropertyValue,
    };
    let Some(members) = snapshot.model_states.as_mut() else {
        return Ok(());
    };
    let key = ModelStatePropertyKey {
        set: property.set.clone(),
        name: property.name.clone(),
    };
    if members.property_table().base(&key).is_none() {
        return Ok(());
    }
    let active = members
        .active()
        .ok_or_else(|| Error("property edit: expected an active member".into()))?
        .id;
    let selected: Vec<u64> = members
        .list()
        .iter()
        .filter(|member| {
            members.edit_scope() == ModelStateEditScope::AllMembers || member.id == active
        })
        .map(|member| member.id)
        .collect();
    let table = members.property_table();
    let mut values = table.values().to_vec();
    values.retain(|value| value.property != key || !selected.contains(&value.member));
    values.extend(selected.into_iter().map(|member| ModelStatePropertyValue {
        member,
        property: key.clone(),
        value: property.value.clone(),
    }));
    let table = ModelStatePropertyTable::new(vec![], values).with_bases(table.bases().to_vec());
    members.set_property_table(table).map_err(Error)
}

#[cfg(test)]
#[path = "property_projection_tests.rs"]
mod tests;
