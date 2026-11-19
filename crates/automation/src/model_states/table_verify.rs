// SPDX-License-Identifier: GPL-2.0-only
//! Verify a table through the same private candidate used by SetTable (#309).
use super::*;
use oblikovati_model::{dependency::DependencyError, parameter::ParameterError};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

pub(crate) fn verify<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: SetModelStateTableArgs,
    geometry: impl FnOnce(u64, &DocumentSnapshot) -> Result<(), String>,
) -> Result<ModelStateTableVerification, Error> {
    let id = workspace
        .active_document()
        .ok_or_else(|| Error("model-state verification: expected an active part".into()))?;
    let session = workspace
        .session(id)
        .map_err(|error| Error(error.to_string()))?;
    if request.expected_generation != session.generation() {
        return Err(Error(format!(
            "model-state table generation {}: expected {}",
            request.expected_generation,
            session.generation()
        )));
    }
    if session.in_transaction() {
        return Err(Error(format!(
            "model-state table document {id}: expected no open transaction"
        )));
    }
    let diagnostics = parameter_diagnostics(session.snapshot(), &request);
    if !diagnostics.is_empty() {
        return Ok(result(diagnostics));
    }
    let candidate = session.preview_projected(
        |snapshot, ids| super::table_apply::apply(snapshot, ids, request),
        |candidate, _| Ok(candidate.clone()),
    );
    let candidate = match candidate {
        Ok(candidate) => candidate,
        Err(error) => return Ok(result(vec![general(error.to_string())])),
    };
    let diagnostics = match geometry(id, &candidate) {
        Ok(()) => vec![],
        Err(error) => vec![general(error)],
    };
    Ok(result(diagnostics))
}
fn result(diagnostics: Vec<ModelStateTableDiagnostic>) -> ModelStateTableVerification {
    ModelStateTableVerification {
        valid: diagnostics.is_empty(),
        diagnostics: Some(diagnostics),
    }
}
fn general(message: String) -> ModelStateTableDiagnostic {
    ModelStateTableDiagnostic {
        row: None,
        parameter_id: 0,
        expression: String::new(),
        message,
    }
}
pub(super) fn parameter_diagnostics(
    snapshot: &DocumentSnapshot,
    request: &SetModelStateTableArgs,
) -> Vec<ModelStateTableDiagnostic> {
    let (Some(rows), Some(columns)) = (&request.rows, &request.parameters) else {
        return vec![];
    };
    let mut diagnostics = Vec::new();
    for row in rows {
        let mut changes = Vec::new();
        for column in columns {
            let Some(cell) = column
                .cells
                .as_deref()
                .unwrap_or_default()
                .iter()
                .find(|cell| cell.row == row.ref_)
            else {
                continue;
            };
            changes.push((
                column.parameter_id,
                cell.expression
                    .as_deref()
                    .cloned()
                    .unwrap_or_else(|| column.base.clone()),
            ));
        }
        let mut parameters = snapshot.parameters.clone();
        if let Err(error) = parameters.set_expressions_detailed(&changes) {
            let mut ids = vec![error.parameter];
            if let ParameterError::Dependency(DependencyError::Cycle { id, driver }) = &error.cause
            {
                ids.push(*id);
                ids.push(*driver);
            }
            ids.sort_unstable();
            ids.dedup();
            let message = error.cause.to_string();
            for id in ids {
                if let Some((_, expression)) =
                    changes.iter().find(|(parameter, _)| *parameter == id)
                {
                    diagnostics.push(ModelStateTableDiagnostic {
                        row: Some(Box::new(row.ref_.clone())),
                        parameter_id: id,
                        expression: expression.clone(),
                        message: message.clone(),
                    });
                }
            }
        }
    }
    diagnostics
}
