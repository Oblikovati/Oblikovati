// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::assets::{
    AppearanceAsset, AssetCollection, BuiltinAssetCatalog, DocumentAssets, MaterialAsset,
    ScopedAssets,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
/// Implemented session catalog reads; creation, update, and assignment are separate capabilities.
pub const ASSET_QUERY_METHODS: [&str; 4] = [
    METHOD_MATERIALS_LIST,
    METHOD_MATERIALS_GET,
    METHOD_APPEARANCES_LIST,
    METHOD_APPEARANCES_GET,
];
enum AssetQuery {
    Materials,
    Appearances,
    Material(AssetRefArgs),
    Appearance(AssetRefArgs),
}
pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let query = classify_asset_query(method, request)?;
    let catalog = files.builtin_catalog().map_err(|cause| {
        Error(format!(
            "asset query {method:?}: expected an admitted catalog: {cause}"
        ))
    })?;
    let empty_document = DocumentAssets::default();
    let document = active_document_assets(files)?.unwrap_or(&empty_document);
    let project = files.active_project().map(|project| &project.assets);
    query_document_catalog(catalog, document, project, query)
}
fn query_document_catalog(
    catalog: &BuiltinAssetCatalog,
    document: &DocumentAssets,
    project: Option<&oblikovati_model::assets::ProjectAssetLibrary>,
    query: AssetQuery,
) -> Result<Vec<u8>, Error> {
    let no_project_materials = AssetCollection::default();
    let no_project_appearances = AssetCollection::default();
    let materials = ScopedAssets {
        document: document.materials(),
        project: project.map_or(&no_project_materials, |assets| assets.materials()),
        builtin: catalog.materials(),
    };
    let appearances = ScopedAssets {
        document: document.appearances(),
        project: project.map_or(&no_project_appearances, |assets| assets.appearances()),
        builtin: catalog.appearances(),
    };
    match query {
        AssetQuery::Materials => list_materials(&materials),
        AssetQuery::Appearances => list_appearances(&appearances),
        AssetQuery::Material(reference) => get_material(&materials, reference),
        AssetQuery::Appearance(reference) => get_appearance(&appearances, reference),
    }
}
fn classify_asset_query(method: &str, request: &[u8]) -> Result<AssetQuery, Error> {
    match method {
        METHOD_MATERIALS_LIST => Ok(AssetQuery::Materials),
        METHOD_APPEARANCES_LIST => Ok(AssetQuery::Appearances),
        METHOD_MATERIALS_GET => Ok(AssetQuery::Material(decode(method, request)?)),
        METHOD_APPEARANCES_GET => Ok(AssetQuery::Appearance(decode(method, request)?)),
        _ => Err(Error(format!(
            "asset query {method:?}: expected one of {ASSET_QUERY_METHODS:?}"
        ))),
    }
}
fn list_materials(materials: &ScopedAssets<'_, MaterialAsset>) -> Result<Vec<u8>, Error> {
    encode(&ListMaterialsResult {
        materials: Some(
            materials
                .display_order()
                .map_err(Error)?
                .into_iter()
                .map(material_info)
                .collect(),
        ),
    })
}
fn list_appearances(appearances: &ScopedAssets<'_, AppearanceAsset>) -> Result<Vec<u8>, Error> {
    encode(&ListAppearancesResult {
        appearances: Some(
            appearances
                .display_order()
                .map_err(Error)?
                .into_iter()
                .map(|asset| asset.info())
                .collect(),
        ),
    })
}
fn get_material(
    materials: &ScopedAssets<'_, MaterialAsset>,
    reference: AssetRefArgs,
) -> Result<Vec<u8>, Error> {
    let asset = materials.resolve(&reference.id).map_err(Error)?;
    encode(&material_info(asset))
}
fn get_appearance(
    appearances: &ScopedAssets<'_, AppearanceAsset>,
    reference: AssetRefArgs,
) -> Result<Vec<u8>, Error> {
    let asset = appearances.resolve(&reference.id).map_err(Error)?;
    encode(&asset.info())
}
fn material_info(asset: &MaterialAsset) -> MaterialInfo {
    use oblikovati_api::contract::Material;
    let mut info = asset.info();
    info.isotropy_class = asset.isotropy_class().0.into_owned();
    info
}

fn active_document_assets<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Option<&DocumentAssets>, Error> {
    files
        .workspace()
        .active_document()
        .map(|id| {
            files
                .workspace()
                .session(id)
                .map(|session| &session.snapshot().assets)
                .map_err(|cause| Error(cause.to_string()))
        })
        .transpose()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn unknown_asset_route_is_classified_before_catalog_access() {
        let refusal = classify_asset_query("invalid.asset.route", b"null")
            .err()
            .unwrap();
        assert!(refusal.to_string().contains("invalid.asset.route"));
    }
}
