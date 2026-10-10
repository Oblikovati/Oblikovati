// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::parameter::{
    DerivedTable, DerivedTableHealth, ParameterCollection, ParameterValue,
};

pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<ListDerivedParameterTablesResult, Error> {
    let parameters = active_parameters(files)?;
    let tables = parameters
        .derived_tables()
        .iter()
        .map(|table| describe(files, parameters, table))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(ListDerivedParameterTablesResult {
        tables: nonempty(tables),
    })
}

fn active_parameters<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<&ParameterCollection<u64>, Error> {
    let id = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("derived tables: expected an active document".into()))?;
    let info = files
        .document_info(id)
        .map_err(|error| Error(error.to_string()))?;
    if !matches!(info.type_.as_str(), "part" | "assembly") {
        return Err(Error(format!(
            "derived tables document type {:?}: expected part or assembly",
            info.type_
        )));
    }
    let parameters = &files
        .workspace()
        .session(id)
        .map_err(|error| Error(error.to_string()))?
        .snapshot()
        .parameters;
    Ok(parameters)
}

fn describe<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    parameters: &ParameterCollection<u64>,
    table: &DerivedTable<u64>,
) -> Result<DerivedParameterTableInfo, Error> {
    let source = files
        .resolve_parameter_source(&table.source_document)
        .map_err(|error| Error(error.to_string()))?;
    let available = source.map(|id| {
        candidate_names(
            &files
                .workspace()
                .session(id)
                .expect("resolved owner")
                .snapshot()
                .parameters,
        )
    });
    Ok(DerivedParameterTableInfo {
        id: table.id,
        source_document: table.source_document.clone(),
        linked: nonempty(table.linked.clone()),
        available: available.and_then(nonempty),
        health: health(table, source.is_some()),
        references: nonempty(
            parameters
                .derived_references(table.id)
                .map_err(|error| Error(error.to_string()))?,
        ),
        has_reference_component: table.owned_by_feature,
        reference_component: if table.owned_by_feature {
            table.source_document.clone()
        } else {
            String::new()
        },
    })
}

fn candidate_names(parameters: &ParameterCollection<u64>) -> Vec<String> {
    parameters
        .ids()
        .iter()
        .filter_map(|id| {
            let parameter = parameters.get(*id)?;
            (parameter.kind == USER_PARAM
                && matches!(parameter.last_good_value, Some(ParameterValue::Numeric(_))))
            .then(|| parameter.name.clone())
        })
        .collect()
}

fn health(table: &DerivedTable<u64>, source_available: bool) -> String {
    match &table.health {
        DerivedTableHealth::Missing(names) => format!("removed at source: [{}]", names.join(" ")),
        DerivedTableHealth::Unavailable => {
            format!("source document {} is unavailable", table.source_document)
        }
        _ if !source_available => {
            format!("source document {} is unavailable", table.source_document)
        }
        DerivedTableHealth::Unresolved => format!(
            "source document {} has not been synchronized",
            table.source_document
        ),
        DerivedTableHealth::Healthy => String::new(),
    }
}

fn nonempty<T>(items: Vec<T>) -> Option<Vec<T>> {
    (!items.is_empty()).then_some(items)
}
