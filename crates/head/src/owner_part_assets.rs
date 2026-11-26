// SPDX-License-Identifier: GPL-2.0-only
//! Project, document, and catalog assets for the part ribbon.
use super::*;
use oblikovati_api::{
    client::Client,
    contract::{Appearance, Material},
    wire::{AssignAppearanceArgs, AssignMaterialArgs},
};
use oblikovati_gui::{
    AppearanceCopy, AppearanceTemplate, GuiAppearanceChange, GuiMaterialChoice, GuiPartAssetKind,
};
use oblikovati_model::assets::{
    AppearanceAsset, AppearanceMetadata, CeramicAppearance, ConcreteAppearance, DocumentAssets,
    MaterialAsset, ScopedAssets,
};

macro_rules! owned_part_asset_work_methods {
    () => {
        fn edit_document_appearance_from_gui(
            &mut self,
            edit: &oblikovati_gui::GuiAppearanceEdit,
        ) -> Result<bool, String> {
            self.edit_document_appearance(edit)
        }
        fn assign_part_asset_from_gui(
            &mut self,
            document: u64,
            kind: oblikovati_gui::GuiPartAssetKind,
            id: &str,
        ) -> Result<bool, String> {
            self.assign_part_asset(document, kind, id)
        }
    };
}
pub(super) use owned_part_asset_work_methods;

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
                    appearance_preview: None,
                    appearance_metadata: None,
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

    pub(super) fn edit_document_appearance(
        &mut self,
        edit: &oblikovati_gui::GuiAppearanceEdit,
    ) -> Result<bool, String> {
        let document = edit.document;
        self.require_browser_document(document)?;
        if self
            .files
            .document_type(document)
            .map_err(|err| err.to_string())?
            != oblikovati_api::types::DOCUMENT_PART
        {
            return Err(format!(
                "appearance edit in document {document}: expected an active Part"
            ));
        }
        let name = match &edit.change {
            GuiAppearanceChange::Create { name, .. }
            | GuiAppearanceChange::Duplicate { name, .. }
            | GuiAppearanceChange::Apply { name, .. } => name,
        };
        if name.trim().is_empty() {
            return Err(format!(
                "appearance name {name:?}: expected a non-empty name"
            ));
        }
        let (created, label) = match &edit.change {
            GuiAppearanceChange::Create { id, name, template } => {
                self.require_new_appearance_id(document, id)?;
                (
                    Some(self.new_document_appearance(id, name, *template)?),
                    "Create document appearance",
                )
            }
            GuiAppearanceChange::Duplicate { id, .. } => {
                self.require_new_appearance_id(document, id)?;
                (None, "Duplicate document appearance")
            }
            GuiAppearanceChange::Apply { .. } => (None, "Edit document appearance"),
        };
        self.files
            .commands(document)
            .map_err(|err| err.to_string())?
            .run(label, |snapshot, _| {
                commit_appearance_edit(&mut snapshot.assets, &edit.change, created.as_ref())
            })
            .map_err(|err| err.to_string())?;
        Ok(true)
    }

    fn require_new_appearance_id(&self, document: u64, id: &str) -> Result<(), String> {
        if self
            .gui_appearance_choices(document, None)?
            .iter()
            .any(|asset| asset.id == id)
        {
            return Err(format!("appearance {id:?}: expected a new asset identity"));
        }
        Ok(())
    }

    fn new_document_appearance(
        &self,
        id: &str,
        name: &str,
        template: AppearanceTemplate,
    ) -> Result<AppearanceAsset, String> {
        let catalog = self.files.builtin_catalog()?;
        let base = catalog
            .appearances()
            .get(oblikovati_model::assets::DEFAULT_APPEARANCE_ID)
            .ok_or_else(|| "appearance: expected the neutral built-in definition".to_owned())?;
        let mut asset = base.duplicate(id, name, oblikovati_api::types::ASSET_DOCUMENT)?;
        match template {
            AppearanceTemplate::Ceramic => initialize_typed_surface(
                &mut asset,
                1.,
                CeramicAppearance::default().finish.surface(),
            )?,
            AppearanceTemplate::Concrete => initialize_typed_surface(
                &mut asset,
                120. / 255.,
                ConcreteAppearance::default().surface(),
            )?,
            AppearanceTemplate::Generic => (),
        }
        Ok(asset)
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

fn initialize_typed_surface(
    asset: &mut AppearanceAsset,
    gray: f32,
    [roughness, weight]: [u8; 2],
) -> Result<(), String> {
    let mut record = asset.info();
    [
        record.base.color.r,
        record.base.color.g,
        record.base.color.b,
    ] = [gray; 3];
    record.specular.roughness = f32::from(roughness) / 255.;
    record.specular.weight = f32::from(weight) / 255.;
    asset.edit(record)?;
    Ok(())
}

fn commit_appearance_edit(
    assets: &mut DocumentAssets,
    edit: &GuiAppearanceChange,
    created: Option<&AppearanceAsset>,
) -> Result<bool, String> {
    if let GuiAppearanceChange::Duplicate {
        source,
        id,
        name,
        kind,
    } = edit
    {
        return duplicate_document_appearance(assets, source, id, name, *kind);
    }
    let GuiAppearanceChange::Apply {
        id,
        name,
        color,
        roughness,
        highlights,
        metadata,
    } = edit
    else {
        return embed_new_appearance(assets, edit, created);
    };
    let mut asset = assets
        .appearances()
        .get(id)
        .cloned()
        .ok_or_else(|| format!("appearance {id:?}: expected a document-owned definition"))?;
    let mut record = asset.info();
    record.display_name = name.clone();
    if let Some(color) = color {
        [
            record.base.color.r,
            record.base.color.g,
            record.base.color.b,
        ] = color.map(|value| f32::from(value) / 255.);
    }
    if let Some(value) = roughness {
        record.specular.roughness = f32::from(*value) / 255.;
    }
    if let Some(value) = highlights {
        record.specular.weight = f32::from(*value) / 255.;
    }
    asset.edit(record)?;
    let embedded = assets.embed_appearance(&asset)?;
    if let Some(metadata) = metadata.as_deref() {
        assets.set_appearance_metadata(id, metadata.clone())?;
    }
    Ok(embedded)
}

fn duplicate_document_appearance(
    assets: &mut DocumentAssets,
    source: &str,
    id: &str,
    name: &str,
    kind: AppearanceCopy,
) -> Result<bool, String> {
    let original = assets.appearances().get(source).ok_or_else(|| {
        format!("appearance {source:?}: expected a document-owned source to copy")
    })?;
    let copy = original.duplicate(id, name, oblikovati_api::types::ASSET_DOCUMENT)?;
    let metadata = assets.appearance_metadata(source).cloned();
    let embedded = assets.embed_appearance(&copy)?;
    if let Some(metadata) = metadata {
        let metadata = match kind {
            AppearanceCopy::SameType => metadata,
            AppearanceCopy::Generic => metadata.into_generic(),
        };
        assets.set_appearance_metadata(id, metadata)?;
    }
    Ok(embedded)
}

fn embed_new_appearance(
    assets: &mut DocumentAssets,
    edit: &GuiAppearanceChange,
    created: Option<&AppearanceAsset>,
) -> Result<bool, String> {
    let created =
        created.ok_or_else(|| "appearance: expected a new document definition".to_owned())?;
    let embedded = assets.embed_appearance(created)?;
    if let GuiAppearanceChange::Create { id, template, .. } = edit {
        let metadata = match template {
            AppearanceTemplate::Generic => return Ok(embedded),
            AppearanceTemplate::Ceramic => AppearanceMetadata {
                ceramic: Some(CeramicAppearance::default()),
                ..Default::default()
            },
            AppearanceTemplate::Concrete => AppearanceMetadata {
                concrete: Some(ConcreteAppearance::default()),
                ..Default::default()
            },
        };
        assets.set_appearance_metadata(id, metadata)?;
    }
    Ok(embedded)
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
