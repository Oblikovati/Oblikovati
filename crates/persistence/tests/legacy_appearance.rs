// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::AppearanceInfo;
use oblikovati_persistence::package::DocumentPackage;
use serde::Deserialize;
#[derive(Deserialize)]
struct GoReceipt {
    baseline: String,
    license: String,
    cases: Vec<GoMigrationCase>,
}
#[derive(Deserialize)]
struct GoMigrationCase {
    input: String,
    expected: AppearanceInfo,
}
fn legacy_package(input: &str) -> DocumentPackage {
    let record = serde_yaml_ng::from_str::<serde_yaml_ng::Value>(input).unwrap();
    let model = serde_yaml_ng::Value::Mapping(serde_yaml_ng::Mapping::from_iter([(
        "materials".into(),
        serde_yaml_ng::to_value(serde_yaml_ng::Mapping::from_iter([(
            "appearances".into(),
            serde_yaml_ng::Value::Sequence(vec![record]),
        )]))
        .unwrap(),
    )]));
    let mut root = serde_yaml_ng::Mapping::new();
    root.insert("schemaVersion".into(), 2u32.into());
    root.insert("documentType".into(), 1u32.into());
    root.insert("model".into(), model);
    DocumentPackage::decode(serde_yaml_ng::to_string(&root).unwrap().as_bytes()).unwrap()
}
#[test]
fn full_eight_bit_domain_and_emission_split_match_the_independent_go_migration() {
    let receipt: GoReceipt =
        serde_yaml_ng::from_str(include_str!("legacy_appearance/go-baseline.json")).unwrap();
    assert_eq!(receipt.baseline, "aa8297d1b0ac33c024f19e54821536d76e8a447a");
    assert_eq!(receipt.license, "SPDX-License-Identifier: GPL-2.0-only");
    assert_eq!(receipt.cases.len(), 256);
    for case in receipt.cases {
        let assets = legacy_package(&case.input).restore_assets().unwrap();
        assert_eq!(
            assets.appearances().get(&case.expected.id).unwrap().info(),
            case.expected
        );
    }
}
#[test]
fn missing_or_black_emission_leaves_every_unrepresented_lobe_off() {
    for emissive in ["", "emissive: '#000000ff'\n"] {
        let package = legacy_package(&format!(
            "id: old\nname: Old\nalbedo: '#804020ff'\nmetallic: 0.3\nroughness: 0.6\nopacity: 1\n{emissive}"
        ));
        let assets = package.restore_assets().unwrap();
        let look = assets.appearances().get("old").unwrap().info();
        assert_eq!(look.emission.luminance, 0.);
        assert_eq!(look.emission.color, Default::default());
        assert_eq!(look.transmission, Default::default());
        assert_eq!(look.subsurface, Default::default());
        assert_eq!(look.coat, Default::default());
        assert_eq!(look.fuzz, Default::default());
        assert_eq!(look.thin_film, Default::default());
        assert_eq!(look.base.metalness, 0.3);
        assert_eq!(look.specular.roughness, 0.6);
        assert_eq!(look.specular.weight, 1.);
    }
}
#[test]
fn first_capture_upgrades_legacy_shape_and_keeps_extension_fields() {
    let original = legacy_package(
        "id: old\nname: Old\nalbedo: '#80402080'\nmetallic: 0.3\nroughness: 0.6\nopacity: 0.75\nfuture: retained\n",
    );
    let assets = original.restore_assets().unwrap();
    let saved = original.with_assets(&assets).unwrap();
    let raw: serde_yaml_ng::Value = serde_yaml_ng::from_slice(&saved.encode().unwrap()).unwrap();
    let look = &raw["model"]["materials"]["appearances"][0];
    assert!(look.get("albedo").is_none());
    assert!(look.get("base").is_some());
    assert_eq!(look["future"], "retained");
    assert_eq!(saved.restore_assets().unwrap(), assets);
    assert_eq!(saved.with_assets(&assets).unwrap(), saved);
}
#[test]
fn invalid_colors_scalars_and_mixed_shapes_are_named_declines() {
    for tail in [
        "albedo: invalid",
        "albedo: '#gg0000'",
        "albedo: '#00000α'",
        "albedo: '#0000000g'",
        "albedo: '#ffffff'\nemissive: bad",
        "albedo: '#ffffff'\nmetallic: .nan",
        "albedo: '#ffffff'\nroughness: 2",
        "albedo: '#ffffff'\nbase: {}",
        "metallic: 0.5",
        "albedo: '#ffffff'\nsource: builtin",
    ] {
        let package = legacy_package(&format!("id: bad\nname: Bad\n{tail}\n"));
        assert!(
            package.restore_assets().is_err(),
            "input {tail}: expected classified migration refusal"
        );
    }
}
