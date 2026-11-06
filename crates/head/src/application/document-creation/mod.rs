// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::{Caller, Client},
    wire::{ActivateDocumentArgs, CloseDocumentArgs, CreateDocumentArgs, SetDocumentUnitsArgs},
};
use oblikovati_gui::GuiDocumentTemplate;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn create_template_document(
        &mut self,
        template: GuiDocumentTemplate,
    ) -> Result<bool, String> {
        let state = self.gui_command_state()?;
        if state.transaction_open || state.authoring == oblikovati_gui::GuiAuthoringContext::Sketch
        {
            return Err(format!(
                "document template {template:?}: expected no active edit or transaction"
            ));
        }
        let previous = self.files.workspace().active_document();
        let name = self.next_new_document_name(template.name_prefix());
        create_template(
            &Client::new(self.document_host()),
            template,
            &name,
            previous,
        )?;
        Ok(true)
    }
}
fn create_template<T: Caller>(
    client: &Client<T>,
    template: GuiDocumentTemplate,
    name: &str,
    previous: Option<u64>,
) -> Result<(), String> {
    let created = client
        .documents()
        .create(&CreateDocumentArgs {
            type_: template.type_name().into(),
            name: name.into(),
            sub_type: template.sub_type().into(),
        })
        .map_err(|cause| {
            format!("document template {template:?}: expected a new document: {cause}")
        })?;
    let result = client.documents().set_units(&SetDocumentUnitsArgs {
        length_unit: Some(Box::new(template.length_unit().into())),
        ..Default::default()
    });
    if let Err(cause) = result {
        rollback_template(client, created.id, previous)?;
        return Err(format!(
            "document template {template:?}: expected length unit {:?}: {cause}",
            template.length_unit()
        ));
    }
    Ok(())
}
fn rollback_template<T: Caller>(
    client: &Client<T>,
    created: u64,
    previous: Option<u64>,
) -> Result<(), String> {
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: created,
            force: true,
        })
        .map_err(|cause| {
            format!("failed template document {created}: expected rollback close: {cause}")
        })?;
    if let Some(id) = previous {
        client
            .documents()
            .activate(&ActivateDocumentArgs { id })
            .map_err(|cause| {
                format!("previous document {id}: expected rollback activation: {cause}")
            })?;
    }
    Ok(())
}
