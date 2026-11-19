// SPDX-License-Identifier: GPL-2.0-only
//! Typed projection of one document-owned mixed model-state table.
use super::*;
use oblikovati_api::types::{
    MODEL_STATE_TABLE_PARAMETER_COLUMN, MODEL_STATE_TABLE_PROPERTY_COLUMN,
    MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
};
use oblikovati_model::model_state::ModelStateTableColumn;
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

pub(crate) fn read<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<ModelStateTableView, Error> {
    let id = workspace
        .active_document()
        .ok_or_else(|| Error("model-state table: expected an active part".into()))?;
    let session = workspace
        .session(id)
        .map_err(|cause| Error(cause.to_string()))?;
    view(session.snapshot(), session.generation())
}

pub(super) fn view(
    snapshot: &DocumentSnapshot,
    generation: u64,
) -> Result<ModelStateTableView, Error> {
    let members = snapshot
        .model_states
        .as_ref()
        .ok_or_else(|| Error("model-state table: expected an active part".into()))?;
    let active = members
        .active()
        .ok_or_else(|| Error("model-state table: expected an active member".into()))?
        .id;
    Ok(ModelStateTableView {
        generation,
        active_member_id: active,
        rows: Some(
            members
                .list()
                .iter()
                .map(|member| ModelStateTableRow {
                    ref_: row_ref(member.id),
                    name: member.name.clone(),
                    source: None,
                })
                .collect(),
        ),
        columns: Some(members.table_columns().iter().map(column_ref).collect()),
        properties: Some(super::table_values::properties(members)),
        suppressions: Some(super::table_values::suppressions(snapshot, members)?),
        parameters: Some(super::table_values::parameters(members)),
        available_properties: Some(super::table_choices::properties(snapshot, members)),
        available_features: Some(super::table_choices::features(snapshot)),
        available_parameters: Some(super::table_choices::parameters(snapshot)),
        evaluations: Some(super::table_values::evaluations(snapshot, members)?),
    })
}

pub(super) fn row_ref(id: u64) -> ModelStateTableRowRef {
    ModelStateTableRowRef {
        member_id: id,
        draft_key: String::new(),
    }
}
fn column_ref(column: &ModelStateTableColumn) -> ModelStateTableColumnRef {
    match column {
        ModelStateTableColumn::Property(key) => ModelStateTableColumnRef {
            kind: MODEL_STATE_TABLE_PROPERTY_COLUMN,
            property: Some(Box::new(key.clone())),
            ..Default::default()
        },
        ModelStateTableColumn::Suppression(id) => ModelStateTableColumnRef {
            kind: MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
            feature_id: *id,
            ..Default::default()
        },
        ModelStateTableColumn::Parameter(id) => ModelStateTableColumnRef {
            kind: MODEL_STATE_TABLE_PARAMETER_COLUMN,
            parameter_id: *id,
            ..Default::default()
        },
    }
}
