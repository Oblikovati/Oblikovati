// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::{Caller, Client},
    wire::{ActivateDocumentArgs, CloseDocumentArgs, CreateDocumentArgs, SetDocumentUnitsArgs},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind};
mod project_catalog;
pub(super) use project_catalog::ProjectTemplateCatalog;
pub(super) use template_material::project_material;
mod template_material;
use oblikovati_gui::GuiTemplateUnits;
pub(crate) use template_material::gui_template_materials;
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
        let created = create_template(
            &Client::new(self.document_host()),
            template,
            &name,
            previous,
        )?;
        if template.type_name() == "part" {
            self.enter_new_part_startup(created, previous)?;
        }
        Ok(true)
    }
}
/// Measure what each template creates by encoding it: built-in templates have no file on disk.
/// Example: `gui_template_sizes()` for the chooser's Details view.
pub(crate) fn gui_template_sizes() -> Result<Vec<(GuiTemplateKind, u64)>, String> {
    static SIZES: std::sync::OnceLock<Result<Vec<(GuiTemplateKind, u64)>, String>> =
        std::sync::OnceLock::new();
    SIZES
        .get_or_init(|| {
            TEMPLATE_KINDS
                .into_iter()
                .map(encoded_template_size)
                .collect()
        })
        .clone()
}
pub(super) fn template_size_snapshot() -> (Vec<(GuiTemplateKind, u64)>, Option<String>) {
    template_size_result(gui_template_sizes())
}
fn template_size_result(
    result: Result<Vec<(GuiTemplateKind, u64)>, String>,
) -> (Vec<(GuiTemplateKind, u64)>, Option<String>) {
    match result {
        Ok(sizes) => (sizes, None),
        Err(cause) => (Vec::new(), Some(cause)),
    }
}
const TEMPLATE_KINDS: [GuiTemplateKind; 5] = [
    GuiTemplateKind::Part,
    GuiTemplateKind::SheetMetal,
    GuiTemplateKind::Assembly,
    GuiTemplateKind::Drawing,
    GuiTemplateKind::Presentation,
];
fn encoded_template_size(kind: GuiTemplateKind) -> Result<(GuiTemplateKind, u64), String> {
    let template = GuiDocumentTemplate {
        kind,
        ..Default::default()
    };
    let request = template_request(template, template.name_prefix());
    let package =
        oblikovati_automation::creation_package(&request).map_err(|cause| cause.to_string())?;
    let bytes = package.encode().map_err(|cause| cause.to_string())?.len() as u64;
    Ok((kind, bytes))
}
fn create_template<T: Caller>(
    client: &Client<T>,
    template: GuiDocumentTemplate,
    name: &str,
    previous: Option<u64>,
) -> Result<u64, String> {
    let created = client
        .documents()
        .create(&template_request(template, name))
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
    Ok(created.id)
}
fn template_request(template: GuiDocumentTemplate, name: &str) -> CreateDocumentArgs {
    CreateDocumentArgs {
        type_: template.type_name().into(),
        name: name.into(),
        sub_type: template.sub_type().into(),
    }
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
#[cfg(test)]
#[path = "../../../tests/application/document-creation/rollback.rs"]
mod rollback_tests;
