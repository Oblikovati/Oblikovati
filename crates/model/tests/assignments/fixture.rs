// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    contract::Appearance,
    wire::{AppearanceInfo, MaterialInfo},
};
use oblikovati_model::assets::*;
use serde::Deserialize;
#[derive(Deserialize)]
struct GoAssetRecords {
    materials: Vec<MaterialInfo>,
    appearances: Vec<AppearanceInfo>,
}
pub struct AssignmentFixture {
    pub materials: [AssetCollection<MaterialAsset>; 3],
    pub appearances: [AssetCollection<AppearanceAsset>; 3],
}
impl AssignmentFixture {
    pub fn new() -> Self {
        let records: GoAssetRecords = serde_json::from_str(include_str!(
            "../../../persistence/tests/catalog/go-baseline.json"
        ))
        .unwrap();
        let mut materials = AssetCollection::default();
        let mut appearances = AssetCollection::default();
        for record in records.materials {
            materials.insert(MaterialAsset::from_info(record).unwrap());
        }
        for record in records.appearances {
            appearances.insert(AppearanceAsset::from_info(record).unwrap());
        }
        Self {
            materials: [Default::default(), Default::default(), materials],
            appearances: [Default::default(), Default::default(), appearances],
        }
    }
    pub fn material_scopes(&self) -> ScopedAssets<'_, MaterialAsset> {
        ScopedAssets {
            document: &self.materials[0],
            project: &self.materials[1],
            builtin: &self.materials[2],
        }
    }
    pub fn appearance_scopes(&self) -> ScopedAssets<'_, AppearanceAsset> {
        ScopedAssets {
            document: &self.appearances[0],
            project: &self.appearances[1],
            builtin: &self.appearances[2],
        }
    }
    pub fn visual_id(
        &self,
        assignments: &AssetAssignments,
        body: &AssignmentReference,
        face: Option<&AssignmentReference>,
    ) -> String {
        assignments
            .resolve_appearance(
                &self.appearance_scopes(),
                &self.material_scopes(),
                body,
                face,
            )
            .unwrap()
            .id()
    }
}
pub fn reference(key: &str, occurrence: &str) -> AssignmentReference {
    let path = if occurrence.is_empty() {
        vec![]
    } else {
        vec![occurrence.into()]
    };
    AssignmentReference::new(key, path).unwrap()
}
