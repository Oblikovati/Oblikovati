// SPDX-License-Identifier: GPL-2.0-only
use super::support::appearance_record;
use oblikovati_api::{
    contract::Appearance,
    types::{ASSET_BUILTIN, ASSET_DOCUMENT, ASSET_PROJECT},
};
use oblikovati_model::assets::{AppearanceAsset, AssetIdentity};
#[test]
fn visual_asset_satisfies_all_nine_released_openpbr_group_accessors() {
    let expected = appearance_record(ASSET_PROJECT);
    let asset = AppearanceAsset::from_info(expected.clone()).unwrap();
    let contract: &dyn Appearance = &asset;
    assert_eq!(contract.id(), expected.id);
    assert_eq!(asset.asset_id(), expected.id);
    assert_eq!(contract.display_name(), expected.display_name);
    assert_eq!(contract.source(), ASSET_PROJECT);
    assert_eq!(contract.base(), expected.base);
    assert_eq!(contract.specular(), expected.specular);
    assert_eq!(contract.transmission(), expected.transmission);
    assert_eq!(contract.subsurface(), expected.subsurface);
    assert_eq!(contract.coat(), expected.coat);
    assert_eq!(contract.fuzz(), expected.fuzz);
    assert_eq!(contract.thin_film(), expected.thin_film);
    assert_eq!(contract.emission(), expected.emission);
    assert_eq!(contract.geometry(), expected.geometry);
    assert_eq!(asset.info(), expected);
}
#[test]
fn visual_copy_retains_every_property_and_edits_do_not_alias_vectors() {
    let base = AppearanceAsset::from_info(appearance_record(ASSET_BUILTIN)).unwrap();
    let mut copy = base.duplicate("custom", "Custom", ASSET_DOCUMENT).unwrap();
    let mut edited = copy.info();
    edited.id = "foreign".into();
    edited.source = "builtin".into();
    edited.coat.weight = 0.75;
    edited.geometry.normal.as_mut().unwrap().x = 0.9;
    assert!(copy.edit(edited).unwrap());
    assert_eq!(copy.id(), "custom");
    assert_eq!(copy.source(), ASSET_DOCUMENT);
    assert_eq!(copy.coat().weight, 0.75);
    assert_eq!(base.coat().weight, 0.625);
    assert_eq!(base.geometry().normal.unwrap().x, 0.1);
    assert!(base.duplicate("x", "", ASSET_PROJECT).is_err());
}
#[test]
fn builtin_visual_edits_are_inert_and_nonfinite_custom_edits_are_atomic() {
    let mut builtin = AppearanceAsset::from_info(appearance_record(ASSET_BUILTIN)).unwrap();
    let before = builtin.info();
    let mut invalid = before.clone();
    invalid.emission.luminance = f32::NAN;
    assert!(!builtin.edit(invalid.clone()).unwrap());
    assert_eq!(builtin.info(), before);
    let mut custom = AppearanceAsset::from_info(appearance_record(ASSET_PROJECT)).unwrap();
    let original = custom.info();
    assert!(custom.edit(invalid).is_err());
    assert_eq!(custom.info(), original);
}
#[test]
fn identity_origin_and_nested_vector_failures_are_named() {
    let base = appearance_record(ASSET_PROJECT);
    for id in ["", "bad\0id"] {
        let mut invalid = base.clone();
        invalid.id = id.into();
        assert!(
            AppearanceAsset::from_info(invalid)
                .unwrap_err()
                .contains("identity")
        );
    }
    let mut origin = base.clone();
    origin.source = "other".into();
    assert!(
        AppearanceAsset::from_info(origin)
            .unwrap_err()
            .contains("other")
    );
    let mut vector = base;
    vector.geometry.normal.as_mut().unwrap().z = f64::INFINITY;
    assert!(
        AppearanceAsset::from_info(vector)
            .unwrap_err()
            .contains("finite")
    );
}

#[test]
fn visual_api_snapshot_round_trips_every_lobe_and_optional_normal() {
    let expected = appearance_record(ASSET_DOCUMENT);
    let encoded = serde_json::to_vec(&expected).unwrap();
    let loaded = serde_json::from_slice(&encoded).unwrap();
    let asset = AppearanceAsset::from_info(loaded).unwrap();
    assert_eq!(asset.info(), expected);
    assert!(asset.geometry().normal.is_some());
    assert!(asset.geometry().coat_tangent.is_some());
}
