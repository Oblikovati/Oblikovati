// SPDX-License-Identifier: GPL-2.0-only
//! Project and record member-specific parameter expressions through the document recipe.
use super::*;
use oblikovati_model::model_state::{
    ModelStateEditScope, ModelStateParameterTable, ModelStateParameterValue,
};

pub(super) fn project(snapshot: &mut DocumentSnapshot) -> Result<(), Error> {
    let Some(members) = snapshot.model_states.as_ref() else {
        return Ok(());
    };
    let active = members
        .active()
        .ok_or_else(|| Error("model-state parameters: expected an active member".into()))?
        .id;
    let mut changes = Vec::new();
    for base in members.parameter_table().bases() {
        let expression = members
            .parameter_table()
            .expression(active, base.parameter)
            .ok_or_else(|| {
                Error(format!(
                    "model-state parameter {}: expected an inherited expression",
                    base.parameter
                ))
            })?;
        let current = snapshot.parameters.get(base.parameter).ok_or_else(|| {
            Error(format!(
                "model-state parameter {}: expected an existing parameter",
                base.parameter
            ))
        })?;
        if current.source != expression {
            changes.push((base.parameter, expression.to_owned()));
        }
    }
    if changes.is_empty() {
        return Ok(());
    }
    snapshot
        .parameters
        .set_expressions(&changes)
        .map_err(|cause| Error(cause.to_string()))
}

pub(crate) fn record_edits(snapshot: &mut DocumentSnapshot) -> Result<(), Error> {
    let Some(members) = snapshot.model_states.as_mut() else {
        return Ok(());
    };
    let active = members
        .active()
        .ok_or_else(|| Error("model-state parameters: expected an active member".into()))?
        .id;
    let selected: Vec<u64> = members
        .list()
        .iter()
        .filter(|member| {
            members.edit_scope() == ModelStateEditScope::AllMembers || member.id == active
        })
        .map(|member| member.id)
        .collect();
    let table = members.parameter_table();
    let mut values = table.values().to_vec();
    let mut bases = table.bases().to_vec();
    let mut removed = Vec::new();
    for base in table.bases() {
        let Some(parameter) = snapshot.parameters.get(base.parameter) else {
            bases.retain(|item| item.parameter != base.parameter);
            removed.push(base.parameter);
            values.retain(|item| item.parameter != base.parameter);
            continue;
        };
        if table.expression(active, base.parameter) != Some(parameter.source.as_str()) {
            values.retain(|value| {
                value.parameter != base.parameter || !selected.contains(&value.member)
            });
            values.extend(selected.iter().map(|member| ModelStateParameterValue {
                member: *member,
                parameter: base.parameter,
                expression: parameter.source.clone(),
            }));
        }
    }
    let order = members.table_columns().iter().filter(|column| !matches!(column, oblikovati_model::model_state::ModelStateTableColumn::Parameter(id) if removed.contains(id))).cloned().collect();
    members
        .set_parameter_table(ModelStateParameterTable::new(bases, values))
        .map_err(Error)?;
    members.set_table_columns(order).map_err(Error)
}
