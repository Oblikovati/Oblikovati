// SPDX-License-Identifier: GPL-2.0-only
use super::oracle::go_catalog_receipt;
use oblikovati_api::contract::{Appearance, Material};
use oblikovati_persistence::assets::load_builtin_catalog;
#[test]
fn all_shipped_properties_and_display_order_match_the_fixed_go_catalog() {
    let expected = go_catalog_receipt();
    assert_eq!(
        expected.baseline,
        "aa8297d1b0ac33c024f19e54821536d76e8a447a"
    );
    assert_eq!(expected.license, "SPDX-License-Identifier: GPL-2.0-only");
    assert_eq!(expected.materials.len(), 126);
    assert_eq!(expected.appearances.len(), 94);
    let actual = load_builtin_catalog().unwrap();
    assert_eq!(actual.materials().insertion_order().count(), 126);
    assert_eq!(actual.appearances().insertion_order().count(), 94);
    for (asset, expected) in actual
        .materials()
        .insertion_order()
        .zip(&expected.materials)
    {
        let mut info = asset.info();
        info.isotropy_class = asset.isotropy_class().0.into_owned();
        assert_eq!(&info, expected, "physical asset {}", asset.id());
    }
    for (asset, expected) in actual
        .appearances()
        .insertion_order()
        .zip(&expected.appearances)
    {
        assert_eq!(&asset.info(), expected, "visual asset {}", asset.id());
    }
    assert_eq!(expected.sources.len(), 7);
    assert!(expected.sources.iter().all(|source| {
        source.source.starts_with("model/material/catalog/") && source.sha256.len() == 64
    }));
}
#[test]
fn shipped_catalog_is_read_only_and_keeps_required_asset_links() {
    let catalog = load_builtin_catalog().unwrap();
    for id in [
        "steel",
        "aluminum-6061",
        "oak-red",
        "electrical-steel-m270",
        "magnet-ndfeb-n42",
        "magnet-smco-2-17",
        "magnet-ferrite-y30",
    ] {
        let mut physical = catalog.materials().get(id).unwrap().clone();
        let before = physical.info();
        let mut edited = before.clone();
        edited.density = 999.;
        assert!(!physical.edit(edited).unwrap());
        assert_eq!(physical.info(), before);
        let visual = catalog
            .appearances()
            .get(&physical.appearance_id())
            .unwrap();
        assert!(visual.geometry().opacity > 0.);
    }
}
#[test]
fn neutral_appearance_uses_the_go_asset_defaults_instead_of_zero_struct_defaults() {
    let catalog = load_builtin_catalog().unwrap();
    let neutral = catalog.appearances().get("default").unwrap();
    assert_eq!(neutral.base().color.r.to_bits(), 0.702f32.to_bits());
    assert_eq!(neutral.base().weight, 1.);
    assert_eq!(neutral.specular().ior, 1.5);
    assert_eq!(neutral.geometry().opacity, 1.);
}
