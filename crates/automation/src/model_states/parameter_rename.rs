// SPDX-License-Identifier: GPL-2.0-only
//! Rewrite every retained member expression through the canonical parameter rename.
use super::*;
use oblikovati_model::model_state::{
    ModelStateParameterBase, ModelStateParameterTable, ModelStateParameterValue,
};
use oblikovati_model::parameter::ParameterCollection;

pub(crate) fn plan(
    snapshot: &DocumentSnapshot,
    request: &ParameterRenameArgs,
) -> Result<Option<ModelStateParameterTable>, Error> {
    let Some(members) = snapshot.model_states.as_ref() else {
        return Ok(None);
    };
    let table = members.parameter_table();
    if table.bases().is_empty() {
        return Ok(None);
    }
    let target = snapshot.parameters.find(&request.name).ok_or_else(|| {
        Error(format!(
            "parameter {:?}: expected an existing parameter",
            request.name
        ))
    })?;
    let base_edits = table
        .bases()
        .iter()
        .map(|base| (base.parameter, base.expression.clone()))
        .collect::<Vec<_>>();
    let bases = renamed_expressions(&snapshot.parameters, target, &request.new_name, &base_edits)?;
    let bases = bases
        .into_iter()
        .map(|(parameter, expression)| ModelStateParameterBase {
            parameter,
            expression,
        })
        .collect();
    let mut values = Vec::new();
    for member in members.list() {
        let expressions = table
            .bases()
            .iter()
            .map(|base| {
                (
                    base.parameter,
                    table
                        .expression(member.id, base.parameter)
                        .expect("validated base")
                        .to_owned(),
                )
            })
            .collect::<Vec<_>>();
        let renamed = renamed_expressions(
            &snapshot.parameters,
            target,
            &request.new_name,
            &expressions,
        )?;
        for (parameter, expression) in renamed {
            if table
                .values()
                .iter()
                .any(|value| value.member == member.id && value.parameter == parameter)
            {
                values.push(ModelStateParameterValue {
                    member: member.id,
                    parameter,
                    expression,
                });
            }
        }
    }
    Ok(Some(ModelStateParameterTable::new(bases, values)))
}
fn renamed_expressions(
    parameters: &ParameterCollection<u64>,
    target: u64,
    name: &str,
    expressions: &[(u64, String)],
) -> Result<Vec<(u64, String)>, Error> {
    let mut candidate = parameters.clone();
    candidate
        .set_expressions(expressions)
        .map_err(|cause| Error(cause.to_string()))?;
    candidate
        .rename(target, name)
        .map_err(|cause| Error(cause.to_string()))?;
    expressions
        .iter()
        .map(|(id, _)| {
            candidate
                .get(*id)
                .map(|state| (*id, state.source.clone()))
                .ok_or_else(|| {
                    Error(format!(
                        "parameter {id}: expected an existing parameter after rename"
                    ))
                })
        })
        .collect()
}
