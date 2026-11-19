// SPDX-License-Identifier: GPL-2.0-only
//! Validate inactive member expressions when their authored table changes or loads.
use super::{DocumentSnapshot, HistoryError};

impl DocumentSnapshot {
    pub(super) fn validate_member_parameter_expressions(&self) -> Result<(), HistoryError> {
        let Some(members) = self.model_states.as_ref() else {
            return Ok(());
        };
        if members.parameter_table().bases().is_empty() {
            return Ok(());
        }
        for member in members.list() {
            let mut parameters = self.parameters.clone();
            let mut changes = Vec::new();
            for base in members.parameter_table().bases() {
                let expression = members
                    .parameter_table()
                    .expression(member.id, base.parameter)
                    .ok_or_else(|| {
                        HistoryError::ModelStateMembers(format!(
                            "model-state member {} parameter {}: expected an inherited expression",
                            member.id, base.parameter
                        ))
                    })?;
                let current = parameters.get(base.parameter).ok_or_else(|| {
                    HistoryError::ModelStateMembers(format!(
                        "model-state parameter {}: expected an existing parameter",
                        base.parameter
                    ))
                })?;
                if current.source != expression {
                    changes.push((base.parameter, expression.to_owned()));
                }
            }
            parameters.set_expressions(&changes).map_err(|cause| {
                HistoryError::ModelStateMembers(format!(
                    "model-state member {}: expected evaluable parameter expressions: {cause}",
                    member.id
                ))
            })?;
            for base in members.parameter_table().bases() {
                let state = parameters.get(base.parameter).expect("validated parameter");
                if state.current().is_none() {
                    return Err(HistoryError::ModelStateMembers(format!(
                        "model-state member {} parameter {} health {:?}: expected an evaluable expression",
                        member.id, base.parameter, state.health
                    )));
                }
            }
        }
        Ok(())
    }
}
