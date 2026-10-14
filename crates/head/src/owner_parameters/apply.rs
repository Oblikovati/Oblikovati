// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::{
    GuiParameterAction, GuiParameterRowOperation, GuiParameterTextEdit, GuiParameterTextField,
    GuiParameterToggleField,
};
use oblikovati_model::{
    parameter::{ParameterCollection, ParameterValue},
    quantity::Dimension,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
pub(crate) fn apply_owner_parameter<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &mut FileWorkspace<O, W, F>,
    action: &GuiParameterAction,
) -> Result<bool, String> {
    if let GuiParameterAction::Add(creation) = action {
        return super::create::create_owner_parameter(files, creation);
    }
    if let GuiParameterAction::Choices(edit) = action {
        edit.validate()?;
    }
    if let GuiParameterAction::Tolerance(edit) = action {
        edit.validate()?;
    }
    if let GuiParameterAction::Group(edit) = action {
        edit.validate()?;
    }
    let (document, parameter) = parameter_action_identity(action)?;
    require_parameter_owner(files, document, parameter)?;
    run_parameter_edit(files, document, parameter, action)?;
    refresh_parameter_readers(files, document)?;
    Ok(true)
}
fn run_parameter_edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    parameter: u64,
    action: &GuiParameterAction,
) -> Result<(), String> {
    files
        .commands(document)
        .map_err(|cause| cause.to_string())?
        .run(parameter_action_label(action), |snapshot, reservations| {
            apply_reserved_parameter(&mut snapshot.parameters, reservations, parameter, action)
        })
        .map_err(|cause| cause.to_string())
}
fn apply_reserved_parameter(
    parameters: &mut ParameterCollection<u64>,
    reservations: &mut oblikovati_model::history::RecipeReservations<'_>,
    parameter: u64,
    action: &GuiParameterAction,
) -> Result<(), String> {
    match action {
        GuiParameterAction::Row { operation, .. } => {
            apply_parameter_row(parameters, reservations, parameter, *operation)
        }
        _ => apply_parameter_value(parameters, parameter, action),
    }
}
fn parameter_action_identity(action: &GuiParameterAction) -> Result<(u64, u64), String> {
    match action {
        GuiParameterAction::Row {
            document,
            parameter,
            ..
        }
        | GuiParameterAction::Toggle {
            document,
            parameter,
            ..
        } => Ok((*document, *parameter)),
        GuiParameterAction::Text(edit) => Ok((edit.target.document, edit.target.parameter)),
        GuiParameterAction::Choices(edit) => Ok((edit.target.document, edit.target.parameter)),
        GuiParameterAction::Tolerance(edit) => Ok((edit.target.document, edit.target.parameter)),
        GuiParameterAction::Group(edit) => Ok((edit.target.document, edit.target.parameter)),
        found => Err(format!(
            "GUI parameter row action {found:?}: expected row mutation after document-level dispatch"
        )),
    }
}
fn parameter_action_label(action: &GuiParameterAction) -> &'static str {
    match action {
        GuiParameterAction::Row {
            operation: GuiParameterRowOperation::CopyToUser,
            ..
        } => "Copy Parameter",
        GuiParameterAction::Row {
            operation: GuiParameterRowOperation::Delete,
            ..
        } => "Delete Parameter",
        GuiParameterAction::Text(edit) if edit.target.field == GuiParameterTextField::Name => {
            "Rename Parameter"
        }
        _ => "Edit Parameter",
    }
}
fn require_parameter_owner<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    parameter: u64,
) -> Result<(), String> {
    if files.workspace().active_document() != Some(document) {
        return Err(format!(
            "GUI parameter document {document}: expected current document {:?}",
            files.workspace().active_document()
        ));
    }
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| cause.to_string())?;
    if session.snapshot().parameters.get(parameter).is_none() {
        return Err(format!(
            "GUI parameter ID {parameter} in document {document}: expected a registered parameter"
        ));
    }
    Ok(())
}
fn apply_parameter_value(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    action: &GuiParameterAction,
) -> Result<(), String> {
    match action {
        GuiParameterAction::Link(edit) => Err(format!(
            "GUI link request {}: expected document-level link dispatch",
            edit.request
        )),
        GuiParameterAction::DeleteLinkedTable { table, .. } => Err(format!(
            "GUI linked table {table}: expected document-level table dispatch"
        )),
        GuiParameterAction::Row { operation, .. } => Err(format!(
            "GUI row operation {operation:?}: expected identity-reserving row dispatch"
        )),
        GuiParameterAction::Add(creation) => Err(format!(
            "GUI parameter creation {:?}: expected creation dispatch before row mutation",
            creation.name
        )),
        GuiParameterAction::Text(edit) => apply_parameter_text(parameters, id, edit),
        GuiParameterAction::Choices(edit) => apply_parameter_choices(parameters, id, &edit.choices),
        GuiParameterAction::Tolerance(edit) => apply_parameter_tolerance(parameters, id, edit),
        GuiParameterAction::Group(edit) => apply_parameter_group(parameters, id, &edit.key.0),
        GuiParameterAction::Toggle {
            field: GuiParameterToggleField::Boolean,
            value,
            ..
        } => parameters
            .set_boolean(id, *value)
            .map_err(|cause| cause.to_string()),
        GuiParameterAction::Toggle { field, value, .. } => {
            apply_parameter_flag(parameters, id, *field, *value)
        }
    }
}
fn apply_parameter_text(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    edit: &GuiParameterTextEdit,
) -> Result<(), String> {
    match edit.target.field {
        GuiParameterTextField::Name => parameters
            .rename(id, &edit.value)
            .map_err(|cause| cause.to_string()),
        GuiParameterTextField::Equation => apply_parameter_equation(parameters, id, &edit.value),
        GuiParameterTextField::Comment => {
            let mut annotations = parameters
                .get(id)
                .expect("validated parameter")
                .annotations
                .clone();
            annotations.comment = edit.value.clone();
            parameters
                .set_annotations(id, annotations)
                .map_err(|cause| cause.to_string())
        }
    }
}
fn apply_parameter_equation(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    source: &str,
) -> Result<(), String> {
    let state = parameters.get(id).expect("validated parameter");
    if state
        .choices
        .expressions
        .as_ref()
        .is_some_and(|choices| !choices.is_empty())
    {
        return parameters
            .select_value(id, source)
            .map_err(|cause| cause.to_string());
    }
    if matches!(state.last_good_value, Some(ParameterValue::Text(_))) {
        return parameters
            .set_text(id, source)
            .map_err(|cause| cause.to_string());
    }
    let dimension = match state.last_good_value {
        Some(ParameterValue::Numeric(quantity)) => quantity.dimension,
        _ => Dimension::Unitless,
    };
    parameters
        .set_authored_expression(id, source, dimension)
        .map_err(|cause| cause.to_string())
}
fn apply_parameter_flag(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    field: GuiParameterToggleField,
    value: bool,
) -> Result<(), String> {
    let mut annotations = parameters
        .get(id)
        .expect("validated parameter")
        .annotations
        .clone();
    match field {
        GuiParameterToggleField::Key => annotations.is_key = value,
        GuiParameterToggleField::Export => annotations.exposed_as_property = value,
        GuiParameterToggleField::Boolean => {
            return Err("GUI Boolean annotation field: expected a key or export field".into());
        }
    }
    parameters
        .set_annotations(id, annotations)
        .map_err(|cause| cause.to_string())
}
pub(super) fn refresh_parameter_readers<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
) -> Result<(), String> {
    if files
        .workspace()
        .session(document)
        .map_err(|cause| cause.to_string())?
        .in_transaction()
    {
        return Ok(());
    }
    let report = files
        .refresh_derived_from(document)
        .map_err(|cause| cause.to_string())?;
    if !report.failures.is_empty() || !report.blocked.is_empty() {
        return Err(format!(
            "GUI parameter edit completed in document {document}, failed {:?}, blocked {:?}: expected complete derived refresh",
            report.failures, report.blocked
        ));
    }
    Ok(())
}

fn apply_parameter_row(
    parameters: &mut ParameterCollection<u64>,
    reservations: &mut oblikovati_model::history::RecipeReservations<'_>,
    id: u64,
    operation: GuiParameterRowOperation,
) -> Result<(), String> {
    match operation {
        GuiParameterRowOperation::AddToGroup => Err(format!(
            "GUI group open target {id}: expected local editor dispatch"
        )),
        GuiParameterRowOperation::RemoveFromGroups => parameters
            .remove_from_all_groups(id)
            .map_err(|cause| cause.to_string()),
        GuiParameterRowOperation::ClearValueList => parameters
            .clear_expression_list(id)
            .map_err(|cause| cause.to_string()),
        GuiParameterRowOperation::EditTolerance => Err(format!(
            "GUI tolerance open target {id}: expected local editor dispatch"
        )),
        GuiParameterRowOperation::EditValueList => Err(format!(
            "GUI value-list open target {id}: expected local editor dispatch"
        )),
        GuiParameterRowOperation::Delete => {
            parameters.remove(id).map_err(|cause| cause.to_string())
        }
        GuiParameterRowOperation::CopyToUser => {
            let destination = reservations.allocate().map_err(|cause| cause.to_string())?;
            parameters
                .copy_to_user(id, destination)
                .map_err(|cause| cause.to_string())
        }
    }
}

fn apply_parameter_choices(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    choices: &oblikovati_api::wire::ExpressionListInfo,
) -> Result<(), String> {
    parameters
        .set_expression_list(
            id,
            choices.expressions.clone().unwrap_or_default(),
            choices.allow_custom_values,
        )
        .map_err(|cause| cause.to_string())?;
    parameters
        .set_custom_order(id, choices.custom_order)
        .map_err(|cause| cause.to_string())
}

fn apply_parameter_tolerance(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    edit: &oblikovati_gui::GuiParameterToleranceEdit,
) -> Result<(), String> {
    parameters
        .set_deviation(id, edit.tolerance.upper, edit.tolerance.lower)
        .map_err(|cause| cause.to_string())?;
    parameters
        .set_model_value_type(id, edit.selection)
        .map_err(|cause| cause.to_string())
}

fn apply_parameter_group(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    key: &str,
) -> Result<(), String> {
    if parameters.group(key).is_none() {
        parameters
            .add_group(key, key, "")
            .map_err(|cause| cause.to_string())?;
    }
    parameters
        .add_to_group(id, key)
        .map_err(|cause| cause.to_string())
}
