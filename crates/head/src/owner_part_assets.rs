// SPDX-License-Identifier: GPL-2.0-only
//! Project, document, and catalog assets for the part ribbon.
use super::*;
use oblikovati_api::{
    client::Client,
    contract::{Appearance, Material},
    wire::{AssignAppearanceArgs, AssignMaterialArgs},
};
use oblikovati_gui::{GuiMaterialChoice, GuiPartAssetKind};
use oblikovati_model::assets::{DocumentAssets, MaterialAsset, ScopedAssets};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn gui_material_choices(
        &self,
        document: u64,
    ) -> Result<Vec<GuiMaterialChoice>, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let project = self.files.active_project();
        let catalog = self.files.builtin_catalog()?;
        let document = &session.snapshot().assets;
        let used: std::collections::BTreeSet<_> = document
            .assignments
            .part_material_id()
            .into_iter()
            .chain(document.assignments.body_materials().map(|(_, id)| id))
            .collect();
        let look_up = |id: &str| {
            document
                .appearances()
                .get(id)
                .or_else(|| project?.assets.appearances().get(id))
                .or_else(|| catalog.appearances().get(id))
                .map(appearance_swatch)
        };
        let project_materials = project
            .into_iter()
            .flat_map(|project| project.assets.materials().ordered());
        Ok(collect_choices(
            document
                .materials()
                .ordered()
                .chain(project_materials)
                .chain(catalog.materials().ordered()),
            |asset: &MaterialAsset| {
                let id = asset.id();
                let library = asset_library(
                    project.is_some_and(|p| p.assets.materials().get(&id).is_some()),
                    catalog.materials().get(&id).is_some(),
                );
                GuiMaterialChoice {
                    category: catalog.material_category(&id),
                    document: document.materials().get(&id).is_some() || used.contains(id.as_str()),
                    used: used.contains(id.as_str()),
                    library,
                    id,
                    name: asset.display_name(),
                    swatch: look_up(&asset.appearance_id()),
                }
            },
        ))
    }

    pub(super) fn gui_part_material_appearance(
        &self,
        document: &DocumentAssets,
    ) -> Result<Option<String>, String> {
        let Some(id) = document.assignments.part_material_id() else {
            return Ok(None);
        };
        let empty = Default::default();
        let catalog = self.files.builtin_catalog()?;
        let scopes = ScopedAssets {
            document: document.materials(),
            project: self
                .files
                .active_project()
                .map_or(&empty, |project| project.assets.materials()),
            builtin: catalog.materials(),
        };
        Ok(scopes
            .resolve(id)
            .ok()
            .map(Material::appearance_id)
            .filter(|id| !id.is_empty()))
    }

    pub(super) fn assign_part_asset(
        &mut self,
        document: u64,
        kind: GuiPartAssetKind,
        id: &str,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let document_kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        if document_kind != oblikovati_api::types::DOCUMENT_PART {
            return Err(format!(
                "document {document} type {document_kind:?}: expected a part for asset {id:?}"
            ));
        }
        let client = Client::new(self.document_host());
        match kind {
            GuiPartAssetKind::Material => client.materials().assign(&AssignMaterialArgs {
                body_key: String::new(),
                material_id: id.into(),
            }),
            GuiPartAssetKind::Appearance => client.appearances().assign(&AssignAppearanceArgs {
                scope: "part".into(),
                key: String::new(),
                appearance_id: id.into(),
            }),
        }
        .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
}

pub(super) fn asset_library(
    project: bool,
    shipped: bool,
) -> Option<oblikovati_gui::GuiAssetLibrary> {
    use oblikovati_gui::GuiAssetLibrary;
    if project {
        Some(GuiAssetLibrary::Project)
    } else {
        shipped.then_some(GuiAssetLibrary::Shipped)
    }
}

pub(super) fn appearance_swatch(asset: &oblikovati_model::assets::AppearanceAsset) -> [u8; 3] {
    let color = asset.base().color;
    [color.r, color.g, color.b].map(|channel| (channel * 255.).round() as u8)
}

/// Merge document, project and catalog choices without losing their priority order.
pub(super) fn collect_choices<'a, T: 'a>(
    assets: impl Iterator<Item = &'a T>,
    choice: impl Fn(&T) -> oblikovati_gui::GuiAssetChoice,
) -> Vec<oblikovati_gui::GuiAssetChoice> {
    let mut seen = std::collections::BTreeSet::new();
    assets
        .map(choice)
        .filter(|asset| seen.insert(asset.id.clone()))
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn asset_merge_keeps_the_first_identity_and_source_order() {
        let sources = [
            ("steel", "Document steel"),
            ("wood", "Project wood"),
            ("steel", "Catalog steel"),
        ];
        let result = collect_choices(sources.iter(), |(id, name)| {
            oblikovati_gui::GuiAssetChoice {
                id: (*id).into(),
                name: (*name).into(),
                swatch: None,
                ..Default::default()
            }
        });
        assert_eq!(
            result
                .iter()
                .map(|asset| asset.name.as_str())
                .collect::<Vec<_>>(),
            ["Document steel", "Project wood"]
        );
    }
}
