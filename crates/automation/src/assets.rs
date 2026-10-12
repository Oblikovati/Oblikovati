// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::assets::{BuiltinAssetCatalog, MaterialAsset};
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
    match query {
        AssetQuery::Materials => list_materials(catalog),
        AssetQuery::Appearances => list_appearances(catalog),
        AssetQuery::Material(reference) => get_material(catalog, reference),
        AssetQuery::Appearance(reference) => get_appearance(catalog, reference),
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
fn list_materials(catalog: &BuiltinAssetCatalog) -> Result<Vec<u8>, Error> {
    encode(&ListMaterialsResult {
        materials: Some(
            catalog
                .materials()
                .insertion_order()
                .map(material_info)
                .collect(),
        ),
    })
}
fn list_appearances(catalog: &BuiltinAssetCatalog) -> Result<Vec<u8>, Error> {
    encode(&ListAppearancesResult {
        appearances: Some(
            catalog
                .appearances()
                .insertion_order()
                .map(|asset| asset.info())
                .collect(),
        ),
    })
}
fn get_material(catalog: &BuiltinAssetCatalog, reference: AssetRefArgs) -> Result<Vec<u8>, Error> {
    let asset = catalog.materials().get(&reference.id).ok_or_else(|| {
        Error(format!(
            "materials.get id {:?}: expected a registered catalog material",
            reference.id
        ))
    })?;
    encode(&material_info(asset))
}
fn get_appearance(
    catalog: &BuiltinAssetCatalog,
    reference: AssetRefArgs,
) -> Result<Vec<u8>, Error> {
    let asset = catalog.appearances().get(&reference.id).ok_or_else(|| {
        Error(format!(
            "appearances.get id {:?}: expected a registered catalog appearance",
            reference.id
        ))
    })?;
    encode(&asset.info())
}
fn material_info(asset: &MaterialAsset) -> MaterialInfo {
    use oblikovati_api::contract::Material;
    let mut info = asset.info();
    info.isotropy_class = asset.isotropy_class().0.into_owned();
    info
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
