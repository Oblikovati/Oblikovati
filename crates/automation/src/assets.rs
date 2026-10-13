// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::assets::{
    AppearanceAsset, AssetCollection, DocumentAssets, MaterialAsset, ScopedAssets,
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
    with_scoped_assets(files, |materials, appearances| match query {
        AssetQuery::Materials => list_materials(materials),
        AssetQuery::Appearances => list_appearances(appearances),
        AssetQuery::Material(reference) => get_material(materials, reference),
        AssetQuery::Appearance(reference) => get_appearance(appearances, reference),
    })
}
pub(crate) fn with_scoped_assets<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    T,
>(
    files: &FileWorkspace<O, W, F>,
    action: impl FnOnce(
        &ScopedAssets<'_, MaterialAsset>,
        &ScopedAssets<'_, AppearanceAsset>,
    ) -> Result<T, Error>,
) -> Result<T, Error> {
    let catalog = files
        .builtin_catalog()
        .map_err(|cause| Error(cause.to_string()))?;
    let empty_document = DocumentAssets::default();
    let document = active_document_assets(files)?.unwrap_or(&empty_document);
    let empty_project = oblikovati_model::assets::ProjectAssetLibrary::default();
    let project = files
        .active_project()
        .map_or(&empty_project, |project| &project.assets);
    action(
        &scoped_family(
            document.materials(),
            project.materials(),
            catalog.materials(),
        ),
        &scoped_family(
            document.appearances(),
            project.appearances(),
            catalog.appearances(),
        ),
    )
}
fn scoped_family<'a, T>(
    document: &'a AssetCollection<T>,
    project: &'a AssetCollection<T>,
    builtin: &'a AssetCollection<T>,
) -> ScopedAssets<'a, T> {
    ScopedAssets {
        document,
        project,
        builtin,
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
    material_response(asset)
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

pub(crate) fn material_response(asset: &MaterialAsset) -> Result<Vec<u8>, Error> {
    encode(&material_info(asset))
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
