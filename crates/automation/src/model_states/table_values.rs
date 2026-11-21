// SPDX-License-Identifier: GPL-2.0-only
//! Project retained mixed table values and per-member evaluation results.
use super::*;
use std::collections::BTreeSet;

pub(super) fn properties(members: &ModelStateMembers) -> Vec<ModelStateTableProperty> {
    members
        .property_table()
        .bases()
        .iter()
        .map(|base| ModelStateTableProperty {
            property: base.property.clone(),
            base: base.value.clone().map(Box::new),
            cells: Some(
                members
                    .list()
                    .iter()
                    .map(|member| ModelStateTablePropertyCell {
                        row: super::table_view::row_ref(member.id),
                        value: members
                            .property_table()
                            .value(member.id, &base.property)
                            .cloned()
                            .map(Box::new),
                    })
                    .collect(),
            ),
        })
        .collect()
}
pub(super) fn suppressions(
    snapshot: &DocumentSnapshot,
    members: &ModelStateMembers,
) -> Result<Vec<ModelStateTableSuppression>, Error> {
    let ids: BTreeSet<u64> = members.referenced_features().collect();
    ids.into_iter()
        .map(|id| {
            let feature = snapshot
                .work_geometry
                .get(oblikovati_model::feature::FeatureId(id))
                .map_err(|cause| Error(cause.to_string()))?;
            let cells = members
                .list()
                .iter()
                .map(|member| {
                    Ok(ModelStateTableSuppressionCell {
                        row: super::table_view::row_ref(member.id),
                        suppressed: members
                            .suppression_override(member.id, id)
                            .map_err(Error)?
                            .map(Box::new),
                    })
                })
                .collect::<Result<Vec<_>, Error>>()?;
            Ok(ModelStateTableSuppression {
                feature_id: id,
                base: feature.suppressed(),
                cells: Some(cells),
            })
        })
        .collect()
}
pub(super) fn parameters(members: &ModelStateMembers) -> Vec<ModelStateTableParameter> {
    let table = members.parameter_table();
    table
        .bases()
        .iter()
        .map(|base| ModelStateTableParameter {
            parameter_id: base.parameter,
            base: base.expression.clone(),
            cells: Some(
                members
                    .list()
                    .iter()
                    .map(|member| ModelStateTableParameterCell {
                        row: super::table_view::row_ref(member.id),
                        expression: table
                            .values()
                            .iter()
                            .find(|value| {
                                value.member == member.id && value.parameter == base.parameter
                            })
                            .map(|value| Box::new(value.expression.clone())),
                    })
                    .collect(),
            ),
        })
        .collect()
}
pub(super) fn evaluations(
    snapshot: &DocumentSnapshot,
    members: &ModelStateMembers,
) -> Result<Vec<ModelStateTableEvaluation>, Error> {
    let mut result = Vec::new();
    if members.parameter_table().bases().is_empty() {
        return Ok(result);
    }
    for member in members.list() {
        let mut parameters = snapshot.parameters.clone();
        let changes = members
            .parameter_table()
            .bases()
            .iter()
            .map(|base| {
                (
                    base.parameter,
                    members
                        .parameter_table()
                        .expression(member.id, base.parameter)
                        .expect("validated member expression")
                        .to_owned(),
                )
            })
            .collect::<Vec<_>>();
        parameters
            .set_expressions(&changes)
            .map_err(|cause| Error(cause.to_string()))?;
        for base in members.parameter_table().bases() {
            let info = crate::parameters::information(&parameters, base.parameter)?;
            result.push(ModelStateTableEvaluation {
                parameter_id: base.parameter,
                row: super::table_view::row_ref(member.id),
                value: info.value,
                failure: info.health,
            });
        }
    }
    Ok(result)
}
