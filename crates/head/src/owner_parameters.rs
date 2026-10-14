// SPDX-License-Identifier: GPL-2.0-only
#[path = "owner_parameters/apply.rs"]
pub(super) mod apply;
#[path = "owner_parameters/create.rs"]
mod create;
#[path = "owner_parameters/derived.rs"]
pub(super) mod derived;
use oblikovati_api::client::Client;
use oblikovati_api::types::Tolerance;
use oblikovati_automation::{FileQueryHost, parameter_detail};
use oblikovati_gui::{
    GuiParameterLinkParameter, GuiParameterLinkSource, GuiParameterRow, GuiParameterSnapshot,
};
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
pub(super) fn owner_parameter_snapshot<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Option<GuiParameterSnapshot>, String> {
    let workspace = files.workspace();
    let Some(document) = workspace.active_document() else {
        return Ok(None);
    };
    let session = workspace
        .session(document)
        .map_err(|cause| cause.to_string())?;
    let parameters = &session.snapshot().parameters;
    let rows = parameters
        .ids()
        .iter()
        .map(|id| projected_parameter_row(parameters, document, *id))
        .collect::<Result<Vec<_>, String>>()?;
    let linked_tables = Client::new(FileQueryHost::new(files))
        .parameters()
        .list_derived_tables()
        .map_err(|cause| cause.to_string())?
        .tables
        .unwrap_or_default();
    let link_sources = owner_link_sources(files, document)?;
    Ok(Some(GuiParameterSnapshot {
        document,
        rows,
        linked_tables,
        link_sources,
    }))
}
pub(super) fn owner_link_sources<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    target: u64,
) -> Result<Vec<GuiParameterLinkSource>, String> {
    files
        .workspace()
        .documents()
        .iter()
        .copied()
        .filter(|id| *id != target)
        .map(|id| owner_link_source(files, id))
        .collect::<Result<Vec<_>, _>>()
        .map(|sources| sources.into_iter().flatten().collect())
}
fn owner_link_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
) -> Result<Option<GuiParameterLinkSource>, String> {
    let source_document = files
        .parameter_source_name(document)
        .map_err(|cause| cause.to_string())?;
    if !resolves_exact_source(files, &source_document, document)? {
        return Ok(None);
    }
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| cause.to_string())?;
    let parameters = &session.snapshot().parameters;
    let parameters = parameters
        .ids()
        .iter()
        .filter_map(|id| {
            let state = parameters.get(*id)?;
            (state.kind == oblikovati_api::types::USER_PARAM && state.current_value().is_some())
                .then(|| GuiParameterLinkParameter {
                    parameter: *id,
                    name: state.name.clone(),
                })
        })
        .collect::<Vec<_>>();
    if parameters.is_empty() {
        return Ok(None);
    }
    Ok(Some(GuiParameterLinkSource {
        document,
        source_document,
        display_name: files
            .workspace()
            .display_name(document)
            .map_err(|cause| cause.to_string())?
            .into(),
        parameters,
    }))
}
fn resolves_exact_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    source_document: &str,
    document: u64,
) -> Result<bool, String> {
    match files.resolve_parameter_source(source_document) {
        Ok(source) => Ok(source == Some(document)),
        Err(oblikovati_session::files::FileWorkspaceError::AmbiguousSource(_)) => Ok(false),
        Err(cause) => Err(cause.to_string()),
    }
}

fn projected_parameter_row(
    parameters: &ParameterCollection<u64>,
    document: u64,
    id: u64,
) -> Result<GuiParameterRow, String> {
    let detail = parameter_detail(parameters, id)
        .map_err(|cause| format!("GUI parameter ID {id} in document {document}: {cause}"))?;
    Ok(GuiParameterRow {
        parameter: id,
        detail,
        tolerance_text: owner_tolerance_text(parameters, id)?,
        has_groups: !parameters
            .groups_of(id)
            .map_err(|cause| cause.to_string())?
            .is_empty(),
    })
}

fn owner_tolerance_text(parameters: &ParameterCollection<u64>, id: u64) -> Result<String, String> {
    let state = parameters
        .get(id)
        .ok_or_else(|| format!("GUI tolerance parameter {id}: expected a registered parameter"))?;
    if state.tolerance == Tolerance::default() {
        return Ok(String::new());
    }
    let Some(quantity) = parameters
        .inspection_value(id)
        .map_err(|cause| cause.to_string())?
    else {
        return Ok(String::new());
    };
    formatted_tolerance_band(parameters, quantity.dimension, &state.tolerance)
}
fn formatted_tolerance_band(
    parameters: &ParameterCollection<u64>,
    dimension: Dimension,
    band: &Tolerance,
) -> Result<String, String> {
    let upper = parameters
        .units()
        .format_precise(Quantity::new(band.upper, dimension))
        .map_err(|cause| cause.to_string())?;
    let lower = parameters
        .units()
        .format_precise(Quantity::new(band.lower, dimension))
        .map_err(|cause| cause.to_string())?;
    Ok(format!("{upper} / {lower}"))
}
