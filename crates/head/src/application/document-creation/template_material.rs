// SPDX-License-Identifier: GPL-2.0-only
//! Read material intent from the same recipe that document creation uses.
use super::{GuiDocumentTemplate, GuiTemplateKind, template_request};
use oblikovati_api::contract::Material;
use oblikovati_gui::{GuiTemplateMaterial, GuiTemplateMaterialEntry, GuiTemplateUnits};
use oblikovati_model::assets::{AssetCollection, DocumentAssets, MaterialAsset, ScopedAssets};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

type CachedRecipes = Vec<(GuiDocumentTemplate, Result<DocumentAssets, String>)>;

pub(crate) fn gui_template_materials<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
) -> Vec<GuiTemplateMaterialEntry> {
    static RECIPES: std::sync::OnceLock<CachedRecipes> = std::sync::OnceLock::new();
    material_entries(files, RECIPES.get_or_init(load_builtin_recipes))
}
fn material_entries<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    recipes: &[(GuiDocumentTemplate, Result<DocumentAssets, String>)],
) -> Vec<GuiTemplateMaterialEntry> {
    recipes
        .iter()
        .map(|(template, assets)| {
            let material = match assets {
                Ok(assets) => workspace_material(files, assets),
                Err(cause) => GuiTemplateMaterial::Unavailable {
                    cause: cause.clone(),
                },
            };
            GuiTemplateMaterialEntry {
                template: *template,
                material,
            }
        })
        .collect()
}
fn load_builtin_recipes() -> CachedRecipes {
    let mut entries = Vec::new();
    for kind in [GuiTemplateKind::Part, GuiTemplateKind::SheetMetal] {
        for units in [GuiTemplateUnits::Metric, GuiTemplateUnits::Inch] {
            let template = GuiDocumentTemplate { kind, units };
            entries.push((template, template_assets(template)));
        }
    }
    entries
}
fn template_assets(template: GuiDocumentTemplate) -> Result<DocumentAssets, String> {
    let request = template_request(template, template.name_prefix());
    let package = oblikovati_automation::creation_package(&request)
        .map_err(|cause| format!("template {template:?}: {cause}"))?;
    package
        .restore_assets()
        .map_err(|cause| format!("template {template:?}: {cause}"))
}
fn workspace_material<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    assets: &DocumentAssets,
) -> GuiTemplateMaterial {
    let Some(id) = assets.assignments.part_material_id() else {
        return GuiTemplateMaterial::Unassigned;
    };
    let empty = AssetCollection::default();
    let project = files
        .active_project()
        .map_or(&empty, |project| project.assets.materials());
    let local = recipe_material(assets, project, &empty);
    if !matches!(local, GuiTemplateMaterial::Unresolved { .. }) {
        return local;
    }
    match files.builtin_catalog() {
        Ok(catalog) => recipe_material(assets, project, catalog.materials()),
        Err(cause) => GuiTemplateMaterial::Unavailable {
            cause: format!("material {id:?}: {cause}"),
        },
    }
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
