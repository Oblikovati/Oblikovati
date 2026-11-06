// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::{Caller, Client},
    wire::{ActivateDocumentArgs, CloseDocumentArgs, CreateDocumentArgs, SetDocumentUnitsArgs},
};
use oblikovati_gui::GuiPartTemplate;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn create_template_part(
        &mut self,
        template: GuiPartTemplate,
    ) -> Result<bool, String> {
        let state = self.gui_command_state()?;
        if state.transaction_open || state.authoring == oblikovati_gui::GuiAuthoringContext::Sketch
        {
            return Err(format!(
                "part template {template:?}: expected no active edit or transaction"
            ));
        }
        let previous = self.files.workspace().active_document();
        let name = self.next_new_part_name();
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
    template: GuiPartTemplate,
    name: &str,
    previous: Option<u64>,
) -> Result<(), String> {
    let created = client
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: name.into(),
            ..Default::default()
        })
        .map_err(|cause| format!("part template {template:?}: expected a new part: {cause}"))?;
    let result = client.documents().set_units(&SetDocumentUnitsArgs {
        length_unit: Some(Box::new(template.length_unit().into())),
        ..Default::default()
    });
    if let Err(cause) = result {
        rollback_template(client, created.id, previous)?;
        return Err(format!(
            "part template {template:?}: expected length unit {:?}: {cause}",
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
            format!("failed template part {created}: expected rollback close: {cause}")
        })?;
    if let Some(id) = previous {
        client
            .documents()
            .activate(&ActivateDocumentArgs { id })
            .map_err(|cause| {
                format!("previous part {id}: expected rollback activation: {cause}")
            })?;
    }
    Ok(())
}
