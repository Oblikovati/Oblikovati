// SPDX-License-Identifier: GPL-2.0-only
use super::{GuiCommandState, GuiDocumentTemplate, GuiTemplateKind};
use imgui::Ui;

pub(super) fn material_detail(ui: &Ui, state: &GuiCommandState, selected: GuiDocumentTemplate) {
    if let Some(label) = material_label(state, selected) {
        ui.text(label);
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
    let label = match material.map(|item| &item.material) {
        Some(crate::GuiTemplateMaterial::Unassigned) => "Material: Unassigned".into(),
        Some(crate::GuiTemplateMaterial::Assigned { name, .. }) => format!("Material: {name}"),
        Some(crate::GuiTemplateMaterial::Unresolved { id }) => {
            format!("Material unavailable: {id}")
        }
        None => "Material: unavailable".into(),
    };
    Some(label)
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/template_material.rs"]
mod tests;
