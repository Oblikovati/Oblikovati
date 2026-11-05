// SPDX-License-Identifier: GPL-2.0-only
use super::support::material_record;
use oblikovati_api::{
    contract::Material,
    types::{self, ASSET_BUILTIN, ASSET_DOCUMENT, ASSET_PROJECT},
};
use oblikovati_model::assets::{AssetIdentity, MaterialAsset};
#[test]
fn physical_asset_satisfies_the_released_contract_without_losing_groups() {
    let expected = material_record(ASSET_PROJECT);
    let asset = MaterialAsset::from_info(expected.clone()).unwrap();
    let contract: &dyn Material = &asset;
    assert_eq!(contract.id(), expected.id);
    assert_eq!(asset.asset_id(), expected.id);
    assert_eq!(contract.display_name(), expected.display_name);
    assert_eq!(contract.source(), ASSET_PROJECT);
    assert_eq!(contract.density(), expected.density);
    assert_eq!(contract.appearance_id(), expected.appearance_id);
    assert_eq!(contract.mechanical(), expected.mechanical);
    assert_eq!(contract.thermal(), expected.thermal);
    assert_eq!(contract.electrical(), expected.electrical);
    assert_eq!(contract.magnetic(), expected.magnetic);
    assert_eq!(contract.anisotropic(), expected.anisotropic);
    assert_eq!(contract.isotropy_class(), types::ORTHOTROPIC);
    assert_eq!(asset.info(), expected);
}
#[test]
fn blank_symmetry_is_reported_as_isotropic_without_rewriting_saved_input() {
    let mut record = material_record(ASSET_DOCUMENT);
    record.isotropy_class.clear();
    let asset = MaterialAsset::from_info(record).unwrap();
    assert_eq!(asset.isotropy_class(), types::ISOTROPIC);
    assert_eq!(asset.info().isotropy_class, "");
}
#[test]
fn physical_duplicate_is_independent_and_edits_cannot_replace_identity() {
    let base = MaterialAsset::from_info(material_record(ASSET_BUILTIN)).unwrap();
    let mut copy = base.duplicate("custom", "Custom", ASSET_DOCUMENT).unwrap();
    let mut edited = copy.info();
    edited.id = "foreign".into();
    edited.source = "builtin".into();
    edited.density = 2.7;
    assert!(copy.edit(edited).unwrap());
    assert_eq!(copy.id(), "custom");
    assert_eq!(copy.display_name(), "Custom");
    assert_eq!(copy.source(), ASSET_DOCUMENT);
    assert_eq!(copy.density(), 2.7);
    assert_eq!(base.density(), 7.85);
    assert!(base.duplicate("x", "", ASSET_PROJECT).is_err());
}
#[test]
fn invalid_physical_edit_is_atomic_and_builtin_edits_are_inert() {
    let mut custom = MaterialAsset::from_info(material_record(ASSET_PROJECT)).unwrap();
    let before = custom.info();
    let mut invalid = before.clone();
    invalid.density = -1.;
    assert!(custom.edit(invalid.clone()).is_err());
    assert_eq!(custom.info(), before);
    let mut builtin = MaterialAsset::from_info(material_record(ASSET_BUILTIN)).unwrap();
    let old = builtin.info();
    assert!(!builtin.edit(invalid).unwrap());
    assert_eq!(builtin.info(), old);
}
#[test]
fn malformed_materials_report_identity_and_expected_property_shape() {
    let base = material_record(ASSET_PROJECT);
    let mut density = base.clone();
    density.density = -2.;
    assert!(
        MaterialAsset::from_info(density)
            .unwrap_err()
            .contains("-2")
    );
    let mut symmetry = base.clone();
    symmetry.isotropy_class = "unknown".into();
    assert!(
        MaterialAsset::from_info(symmetry)
            .unwrap_err()
            .contains("unknown")
    );
    let mut nested = base;
    nested.anisotropic.g13 = f64::INFINITY;
    assert!(
        MaterialAsset::from_info(nested)
            .unwrap_err()
            .contains("finite")
    );
}

#[test]
fn physical_api_snapshot_round_trips_sourced_directional_and_magnetic_values() {
    let expected = material_record(ASSET_DOCUMENT);
    let encoded = serde_json::to_vec(&expected).unwrap();
    let loaded = serde_json::from_slice(&encoded).unwrap();
    let asset = MaterialAsset::from_info(loaded).unwrap();
    assert_eq!(asset.info(), expected);
    let mut transversely_isotropic = expected;
    transversely_isotropic.isotropy_class = types::TRANSVERSELY_ISOTROPIC.0.into_owned();
    assert_eq!(
        MaterialAsset::from_info(transversely_isotropic)
            .unwrap()
            .isotropy_class(),
        types::TRANSVERSELY_ISOTROPIC
    );
}
