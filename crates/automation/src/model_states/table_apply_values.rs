// SPDX-License-Identifier: GPL-2.0-only
//! Validate and construct typed member values for a complete mixed table draft.
use super::table_rows::ResolvedRows;
use super::*;
use oblikovati_model::model_state::{
    ModelStateParameterBase, ModelStateParameterTable, ModelStateParameterValue,
    ModelStatePropertyBase, ModelStatePropertyTable, ModelStatePropertyValue,
};

pub(super) fn properties(
    snapshot: &DocumentSnapshot,
    members: &ModelStateMembers,
    rows: &ResolvedRows,
    columns: &[ModelStateTableProperty],
) -> Result<ModelStatePropertyTable, Error> {
    let available = super::property_view::available(members, &snapshot.properties);
    let mut keys = Vec::new();
    let mut bases = Vec::new();
    let mut values = Vec::new();
    for column in columns {
        let key = &column.property;
        if !available.contains(key) || keys.contains(key) {
            return Err(Error(format!(
                "model-state property {key:?}: expected a unique available property"
            )));
        }
        super::property_edit::validate_ownership(snapshot, key)?;
        let expected = super::property_view::base(members, &snapshot.properties, key);
        if !super::property_edit::same_default(&column.base, &expected) {
            return Err(Error(format!(
                "model-state property {key:?}: expected the unchanged inherited base"
            )));
        }
        let cells = column.cells.as_deref().ok_or_else(|| {
            Error(format!(
                "model-state property {key:?}: expected cells for every row"
            ))
        })?;
        validate_rows(rows, cells.iter().map(|cell| &cell.row))?;
        for cell in cells {
            if let Some(value) = &cell.value {
                values.push(ModelStatePropertyValue {
                    member: rows.id(&cell.row)?,
                    property: key.clone(),
                    value: (**value).clone(),
                });
            }
        }
        bases.push(ModelStatePropertyBase {
            property: key.clone(),
            value: column.base.as_deref().cloned(),
        });
        keys.push(key.clone());
    }
    for base in members.property_table().bases() {
        if !keys.contains(&base.property) {
            return Err(Error(format!(
                "model-state property {:?}: expected its retained record",
                base.property
            )));
        }
    }
    Ok(ModelStatePropertyTable::new(keys, values).with_bases(bases))
}

pub(super) fn suppressions(
    snapshot: &DocumentSnapshot,
    members: &mut ModelStateMembers,
    rows: &ResolvedRows,
    columns: &[ModelStateTableSuppression],
) -> Result<(), Error> {
    let mut seen = Vec::new();
    for column in columns {
        let id = column.feature_id;
        let feature = snapshot
            .work_geometry
            .get(oblikovati_model::feature::FeatureId(id))
            .map_err(|cause| Error(cause.to_string()))?;
        if seen.contains(&id) || feature.suppressed() != column.base {
            return Err(Error(format!(
                "model-state feature {id}: expected a unique feature and unchanged suppression base"
            )));
        }
        let cells = column.cells.as_deref().ok_or_else(|| {
            Error(format!(
                "model-state feature {id}: expected cells for every row"
            ))
        })?;
        validate_rows(rows, cells.iter().map(|cell| &cell.row))?;
        for cell in cells {
            members
                .set_suppressed_for(
                    rows.id(&cell.row)?,
                    id,
                    column.base,
                    cell.suppressed.as_deref().copied().unwrap_or(column.base),
                )
                .map_err(Error)?;
        }
        seen.push(id);
    }
    for id in snapshot
        .model_states
        .as_ref()
        .expect("validated part")
        .referenced_features()
    {
        if !seen.contains(&id) {
            return Err(Error(format!(
                "model-state feature {id}: expected its retained record"
            )));
        }
    }
    Ok(())
}

pub(super) fn parameters(
    snapshot: &DocumentSnapshot,
    members: &ModelStateMembers,
    rows: &ResolvedRows,
    columns: &[ModelStateTableParameter],
) -> Result<ModelStateParameterTable, Error> {
    let available = super::table_choices::parameters(snapshot);
    let mut seen = Vec::new();
    let mut bases = Vec::new();
    let mut values = Vec::new();
    for column in columns {
        let id = column.parameter_id;
        let choice = available
            .iter()
            .find(|choice| choice.id == id)
            .ok_or_else(|| {
                Error(format!(
                    "model-state parameter {id}: expected an available editable numeric parameter"
                ))
            })?;
        if seen.contains(&id) || choice.base_expression != column.base {
            return Err(Error(format!(
                "model-state parameter {id}: expected a unique parameter and unchanged inherited expression"
            )));
        }
        let cells = column.cells.as_deref().ok_or_else(|| {
            Error(format!(
                "model-state parameter {id}: expected cells for every row"
            ))
        })?;
        validate_rows(rows, cells.iter().map(|cell| &cell.row))?;
        for cell in cells {
            if let Some(expression) = &cell.expression {
                values.push(ModelStateParameterValue {
                    member: rows.id(&cell.row)?,
                    parameter: id,
                    expression: (**expression).clone(),
                });
            }
        }
        bases.push(ModelStateParameterBase {
            parameter: id,
            expression: column.base.clone(),
        });
        seen.push(id);
    }
    for base in members.parameter_table().bases() {
        if !seen.contains(&base.parameter) {
            return Err(Error(format!(
                "model-state parameter {}: expected its retained record",
                base.parameter
            )));
        }
    }
    Ok(ModelStateParameterTable::new(bases, values))
}
fn validate_rows<'a>(
    rows: &ResolvedRows,
    cells: impl Iterator<Item = &'a ModelStateTableRowRef>,
) -> Result<(), Error> {
    let cells: Vec<_> = cells.collect();
    if cells.len() != rows.references.len()
        || cells
            .iter()
            .zip(&rows.references)
            .any(|(cell, expected)| *cell != expected)
    {
        return Err(Error(format!(
            "model-state cells {cells:?}: expected one cell in table row order {:?}",
            rows.references
        )));
    }
    Ok(())
}
