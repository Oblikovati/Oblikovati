// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::USER_PARAM;
use oblikovati_gui::{GuiParameterCreation, GuiParameterInitialValue};
use oblikovati_model::{history::RecipeReservations, parameter::ParameterCollection};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
pub(super) fn create_owner_parameter<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &mut FileWorkspace<O, W, F>,
    creation: &GuiParameterCreation,
) -> Result<bool, String> {
    if files.workspace().active_document() != Some(creation.document) {
        return Err(format!(
            "GUI parameter creation document {}: expected current document {:?}",
            creation.document,
            files.workspace().active_document()
        ));
    }
    files
        .commands(creation.document)
        .map_err(|cause| cause.to_string())?
        .run("Add Parameter", |snapshot, reservations| {
            insert_creation(&mut snapshot.parameters, reservations, creation)
        })
        .map_err(|cause| cause.to_string())?;
    super::apply::refresh_parameter_readers(files, creation.document)?;
    Ok(true)
}
fn insert_creation(
    parameters: &mut ParameterCollection<u64>,
    reservations: &mut RecipeReservations<'_>,
    creation: &GuiParameterCreation,
) -> Result<(), String> {
    let id = reservations.allocate().map_err(|cause| cause.to_string())?;
    let result = match &creation.value {
        GuiParameterInitialValue::Numeric(source) => {
            parameters.insert_expression(id, &creation.name, USER_PARAM, source)
        }
        GuiParameterInitialValue::Text(text) => parameters.insert_text(id, &creation.name, text),
        GuiParameterInitialValue::Boolean(value) => {
            parameters.insert_boolean(id, &creation.name, *value)
        }
    };
    result.map_err(|cause| cause.to_string())
}
