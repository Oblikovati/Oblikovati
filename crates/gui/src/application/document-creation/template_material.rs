// SPDX-License-Identifier: GPL-2.0-only
use super::{GuiCommandState, GuiDocumentTemplate, GuiTemplateKind, TemplateChoice};
use crate::GuiTemplateMaterial;
use imgui::Ui;

pub(super) fn material_detail(ui: &Ui, state: &GuiCommandState, selected: &TemplateChoice) {
    let label = match selected {
        TemplateChoice::BuiltIn(template) => material_label(state, *template),
        TemplateChoice::Project(template) => template
            .material
            .as_ref()
            .map(|material| material_caption(Some(material))),
    };
    if let Some(label) = label {
        ui.text_wrapped(label);
    }
}
fn material_label(state: &GuiCommandState, selected: GuiDocumentTemplate) -> Option<String> {
    if !matches!(
        selected.kind,
        GuiTemplateKind::Part | GuiTemplateKind::SheetMetal
    ) {
        return None;
    }
    let material = state
        .template_materials
        .iter()
        .find(|item| item.template == selected);
    Some(material_caption(material.map(|item| &item.material)))
}

fn material_caption(material: Option<&GuiTemplateMaterial>) -> String {
    match material {
        Some(GuiTemplateMaterial::Unassigned) => "Material: Unassigned".into(),
        Some(GuiTemplateMaterial::Assigned { name, .. }) => format!("Material: {name}"),
        Some(GuiTemplateMaterial::Unresolved { id }) => format!("Material unavailable: {id}"),
        Some(GuiTemplateMaterial::Unavailable { cause }) => {
            format!("Material unavailable: {cause}")
        }
        None => "Material: unavailable".into(),
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/template_material.rs"]
mod tests;
