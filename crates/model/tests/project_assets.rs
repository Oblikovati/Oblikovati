// SPDX-License-Identifier: GPL-2.0-only
#[path = "assets/support.rs"]
mod support;
use oblikovati_api::types::{ASSET_BUILTIN, ASSET_PROJECT};
use oblikovati_model::assets::{AppearanceAsset, MaterialAsset, ProjectAssetLibrary};
#[test]
fn project_admission_retains_typed_groups_and_rejects_other_owners() {
    let material = MaterialAsset::from_info(support::material_record(ASSET_PROJECT)).unwrap();
    let appearance = AppearanceAsset::from_info(support::appearance_record(ASSET_PROJECT)).unwrap();
    let project =
        ProjectAssetLibrary::from_records(vec![material.clone()], vec![appearance.clone()])
            .unwrap();
    assert_eq!(
        project.materials().get("physical").unwrap().info(),
        material.info()
    );
    assert_eq!(
        project.appearances().get("visual").unwrap().info(),
        appearance.info()
    );
    assert_eq!(project, project.clone());
    assert_ne!(project, ProjectAssetLibrary::default());
    assert!(ProjectAssetLibrary::from_records(vec![material.clone(), material], vec![]).is_err());
    let builtin = AppearanceAsset::from_info(support::appearance_record(ASSET_BUILTIN)).unwrap();
    assert!(
        ProjectAssetLibrary::from_records(vec![], vec![builtin])
            .unwrap_err()
            .contains("project")
    );
}

#[test]
fn project_replacements_are_owned_candidates_and_refuse_builtin_sources() {
    let project = ProjectAssetLibrary::default();
    let material = MaterialAsset::from_info(support::material_record(ASSET_PROJECT)).unwrap();
    let appearance = AppearanceAsset::from_info(support::appearance_record(ASSET_PROJECT)).unwrap();
    let candidate = project
        .with_material(material)
        .unwrap()
        .with_appearance(appearance)
        .unwrap();
    assert!(project.materials().ordered().next().is_none());
    assert_eq!(candidate.materials().ordered().count(), 1);
    assert_eq!(candidate.appearances().ordered().count(), 1);
    let builtin = MaterialAsset::from_info(support::material_record(ASSET_BUILTIN)).unwrap();
    assert!(
        candidate
            .with_material(builtin)
            .unwrap_err()
            .contains("project")
    );
    let builtin = AppearanceAsset::from_info(support::appearance_record(ASSET_BUILTIN)).unwrap();
    assert!(
        candidate
            .with_appearance(builtin)
            .unwrap_err()
            .contains("project")
    );
}
