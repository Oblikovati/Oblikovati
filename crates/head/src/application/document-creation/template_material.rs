// SPDX-License-Identifier: GPL-2.0-only
//! Read material intent from the same recipe that document creation uses.
use super::{GuiDocumentTemplate, GuiTemplateKind, template_request};
use oblikovati_api::contract::Material;
use oblikovati_gui::{GuiTemplateMaterial, GuiTemplateMaterialEntry, GuiTemplateUnits};
use oblikovati_model::assets::{AssetCollection, DocumentAssets, MaterialAsset, ScopedAssets};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn gui_template_materials<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Vec<GuiTemplateMaterialEntry>, String> {
    let mut entries = Vec::new();
    for kind in [GuiTemplateKind::Part, GuiTemplateKind::SheetMetal] {
        for units in [GuiTemplateUnits::Metric, GuiTemplateUnits::Inch] {
            let template = GuiDocumentTemplate { kind, units };
            entries.push(material_entry(files, template)?);
        }
    }
    Ok(entries)
}
fn material_entry<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    template: GuiDocumentTemplate,
) -> Result<GuiTemplateMaterialEntry, String> {
    let request = template_request(template, template.name_prefix());
    let package =
        oblikovati_automation::creation_package(&request).map_err(|cause| cause.to_string())?;
    let assets = package
        .restore_assets()
        .map_err(|cause| cause.to_string())?;
    let material = workspace_material(files, &assets)?;
    Ok(GuiTemplateMaterialEntry { template, material })
}

fn workspace_material<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    assets: &DocumentAssets,
) -> Result<GuiTemplateMaterial, String> {
    if assets.assignments.part_material_id().is_none() {
        return Ok(GuiTemplateMaterial::Unassigned);
    }
    let empty = AssetCollection::default();
    let project = files
        .active_project()
        .map_or(&empty, |project| project.assets.materials());
    let local = recipe_material(assets, project, &empty);
    if !matches!(local, GuiTemplateMaterial::Unresolved { .. }) {
        return Ok(local);
    }
    Ok(recipe_material(
        assets,
        project,
        files.builtin_catalog()?.materials(),
    ))
}
fn recipe_material(
    assets: &DocumentAssets,
    project: &AssetCollection<MaterialAsset>,
    builtin: &AssetCollection<MaterialAsset>,
) -> GuiTemplateMaterial {
    let Some(id) = assets.assignments.part_material_id() else {
        return GuiTemplateMaterial::Unassigned;
    };
    let scopes = ScopedAssets {
        document: assets.materials(),
        project,
        builtin,
    };
    match scopes.resolve(id) {
        Ok(asset) => GuiTemplateMaterial::Assigned {
            id: id.into(),
            name: asset.display_name(),
        },
        Err(_) => GuiTemplateMaterial::Unresolved { id: id.into() },
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/template_material.rs"]
mod tests;
