// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::{AppearanceInfo, MaterialInfo};
use oblikovati_model::assets::{AppearanceAsset, MaterialAsset};
use serde::Deserialize;
#[derive(Deserialize)]
pub struct GoCatalogReceipt {
    pub baseline: String,
    pub license: String,
    pub materials: Vec<MaterialInfo>,
    pub appearances: Vec<AppearanceInfo>,
    pub sources: Vec<SourceReceipt>,
}
#[derive(Deserialize)]
pub struct SourceReceipt {
    pub source: String,
    pub sha256: String,
}
pub fn go_catalog_receipt() -> GoCatalogReceipt {
    serde_yaml_ng::from_str(include_str!("go-baseline.json")).unwrap()
}
pub fn small_catalog_records() -> (Vec<MaterialAsset>, Vec<AppearanceAsset>) {
    let mut oracle = go_catalog_receipt();
    let physical = oracle.materials.remove(0);
    let appearance = oracle
        .appearances
        .into_iter()
        .find(|asset| asset.id == physical.appearance_id)
        .unwrap();
    let neutral = go_catalog_receipt().appearances.remove(0);
    (
        vec![MaterialAsset::from_info(physical).unwrap()],
        vec![
            AppearanceAsset::from_info(neutral).unwrap(),
            AppearanceAsset::from_info(appearance).unwrap(),
        ],
    )
}
