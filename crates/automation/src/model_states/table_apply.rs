// SPDX-License-Identifier: GPL-2.0-only
//! Apply one complete mixed model-state table as a private document candidate.
use super::*;
use oblikovati_api::types::{
    MODEL_STATE_TABLE_PARAMETER_COLUMN, MODEL_STATE_TABLE_PROPERTY_COLUMN,
    MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
};
use oblikovati_model::model_state::ModelStateTableColumn;

pub(crate) fn set(
    owner: &mut impl crate::command_owner::CommandOwner,
    request: SetModelStateTableArgs,
) -> Result<Vec<u8>, Error> {
    let document = owner
        .workspace()
        .active_document()
        .ok_or_else(|| Error("model-state table: expected an active part".into()))?;
    let session = owner
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    if session.in_transaction() {
        return Err(Error(format!(
            "model-state table document {document}: expected no open transaction"
        )));
    }
    if request.expected_generation != session.generation() {
        return Err(Error(format!(
            "model-state table generation {}: expected {}",
            request.expected_generation,
            session.generation()
        )));
    }
    let validation =
        super::table_verify::verify(owner.workspace(), request.clone(), |id, candidate| {
            owner.verify_named_candidate(id, candidate)
        })?;
    if let Some(diagnostic) = validation
        .diagnostics
        .as_deref()
        .unwrap_or_default()
        .first()
    {
        return Err(Error(format!(
            "model-state row {:?}, parameter {} expression {:?}: {}",
            diagnostic.row, diagnostic.parameter_id, diagnostic.expression, diagnostic.message
        )));
    }
    let policy = session
        .owned_checkpoint_prefix()
        .and_then(|(policy, _)| policy);
    owner
        .run_named_prepared(
            document,
            policy,
            |snapshot, ids| apply(snapshot, ids, request).map_err(|error| error.to_string()),
            |candidate, created, generation| {
                let table = super::table_view::view(candidate, generation)
                    .map_err(|error| error.to_string())?;
                crate::encode(&ModelStateTableResult {
                    table,
                    created_rows: Some(created),
                })
                .map_err(|error| error.to_string())
            },
        )
        .map_err(Error)
}
pub(super) fn apply(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    request: SetModelStateTableArgs,
) -> Result<Vec<ModelStateTableCreatedRow>, Error> {
    let original = snapshot.clone();
    let members = original
        .model_states
        .as_ref()
        .ok_or_else(|| Error("model-state table: expected an active part".into()))?;
    let rows = super::table_rows::resolve(required(request.rows, "rows")?, members, ids)?;
    let properties = super::table_apply_values::properties(
        &original,
        members,
        &rows,
        &required(request.properties, "properties")?,
    )?;
    let parameters = super::table_apply_values::parameters(
        &original,
        members,
        &rows,
        &required(request.parameters, "parameters")?,
    )?;
    let mut candidate = members.clone();
    candidate.replace_rows(&rows.edits).map_err(Error)?;
    let suppressions = required(request.suppressions, "suppressions")?;
    super::table_apply_values::suppressions(&original, &mut candidate, &rows, &suppressions)?;
    let feature_ids = suppressions
        .iter()
        .map(|column| column.feature_id)
        .collect::<Vec<_>>();
    candidate.set_property_table(properties).map_err(Error)?;
    candidate.set_parameter_table(parameters).map_err(Error)?;
    let order = decode_columns(required(request.columns, "columns")?, &feature_ids)?;
    candidate.set_table_columns(order).map_err(Error)?;
    snapshot.model_states = Some(candidate);
    super::parameter_projection::project(snapshot)?;
    super::property_projection::project(snapshot)?;
    Ok(rows.created)
}
fn required<T>(value: Option<Vec<T>>, name: &str) -> Result<Vec<T>, Error> {
    value.ok_or_else(|| {
        Error(format!(
            "model-state table {name} null: expected an explicit collection"
        ))
    })
}
fn decode_columns(
    columns: Vec<ModelStateTableColumnRef>,
    feature_ids: &[u64],
) -> Result<Vec<ModelStateTableColumn>, Error> {
    let mut result = Vec::new();
    for column in columns {
        let entry = if column.kind == MODEL_STATE_TABLE_PROPERTY_COLUMN
            && column.feature_id == 0
            && column.parameter_id == 0
        {
            ModelStateTableColumn::Property(*column.property.clone().ok_or_else(|| {
                Error(format!(
                    "model-state column {column:?}: expected a property identity"
                ))
            })?)
        } else if column.kind == MODEL_STATE_TABLE_SUPPRESSION_COLUMN
            && column.property.is_none()
            && column.feature_id != 0
            && column.parameter_id == 0
        {
            if !feature_ids.contains(&column.feature_id) {
                return Err(Error(format!(
                    "model-state column {column:?}: expected a retained feature record"
                )));
            }
            ModelStateTableColumn::Suppression(column.feature_id)
        } else if column.kind == MODEL_STATE_TABLE_PARAMETER_COLUMN
            && column.property.is_none()
            && column.feature_id == 0
            && column.parameter_id != 0
        {
            ModelStateTableColumn::Parameter(column.parameter_id)
        } else {
            return Err(Error(format!(
                "model-state column {column:?}: expected one identity matching its kind"
            )));
        };
        result.push(entry);
    }
    Ok(result)
}
