// SPDX-License-Identifier: GPL-2.0-only
use super::oracle::small_catalog_records;
use oblikovati_api::{contract::Material, types, wire::MaterialInfo};
use oblikovati_model::assets::{AppearanceAsset, BuiltinAssetCatalog, MaterialAsset};
#[test]
fn incomplete_or_duplicate_catalogs_are_refused_before_publication() {
    let (physical, visual) = small_catalog_records();
    assert!(
        BuiltinAssetCatalog::from_records(physical.clone(), visual[1..].to_vec())
            .unwrap_err()
            .contains("default")
    );
    let mut duplicated = physical.clone();
    duplicated.push(physical[0].clone());
    assert!(
        BuiltinAssetCatalog::from_records(duplicated, visual.clone())
            .unwrap_err()
            .contains(&physical[0].id())
    );
    let mut duplicated = visual.clone();
    duplicated.push(visual[0].clone());
    assert!(
        BuiltinAssetCatalog::from_records(physical.clone(), duplicated)
            .unwrap_err()
            .contains("unique")
    );
    let mut missing = physical[0].info();
    missing.appearance_id = "absent".into();
    assert!(
        BuiltinAssetCatalog::from_records(vec![MaterialAsset::from_info(missing).unwrap()], visual)
            .unwrap_err()
            .contains("absent")
    );
}
#[test]
fn builtin_admission_refuses_other_sources_and_blank_names() {
    let (physical, visual) = small_catalog_records();
    let mut foreign = physical[0].info();
    foreign.source = "project".into();
    assert!(
        BuiltinAssetCatalog::from_records(
            vec![MaterialAsset::from_info(foreign).unwrap()],
            visual.clone()
        )
        .unwrap_err()
        .contains("builtin")
    );
    for name in ["", "bad\0name"] {
        let mut bad = visual[0].info();
        bad.display_name = name.into();
        assert!(
            BuiltinAssetCatalog::from_records(
                physical.clone(),
                vec![AppearanceAsset::from_info(bad).unwrap(), visual[1].clone()]
            )
            .unwrap_err()
            .contains("named builtin")
        );
    }
}
#[test]
fn every_required_catalog_property_must_be_positive() {
    let (physical, visual) = small_catalog_records();
    for field in [
        "density",
        "youngsModulus",
        "poissonsRatio",
        "yieldStrength",
        "ultimateTensileStrength",
        "conductivity",
        "expansionCoeff",
        "specificHeat",
        "resistivity",
        "relativePermittivity",
    ] {
        let mut zero = physical[0].info();
        zero_physical_property(&mut zero, field);
        let refused = BuiltinAssetCatalog::from_records(
            vec![MaterialAsset::from_info(zero).unwrap()],
            visual.clone(),
        )
        .unwrap_err();
        assert!(refused.contains(field), "{refused}");
    }
}
fn zero_physical_property(record: &mut MaterialInfo, field: &str) {
    match field {
        "density" => record.density = 0.,
        "youngsModulus" => record.mechanical.youngs_modulus = 0.,
        "poissonsRatio" => record.mechanical.poissons_ratio = 0.,
        "yieldStrength" => record.mechanical.yield_strength = 0.,
        "ultimateTensileStrength" => record.mechanical.ultimate_tensile_strength = 0.,
        "conductivity" => record.thermal.conductivity = 0.,
        "expansionCoeff" => record.thermal.expansion_coeff = 0.,
        "specificHeat" => record.thermal.specific_heat = 0.,
        "resistivity" => record.electrical.resistivity = 0.,
        "relativePermittivity" => record.electrical.relative_permittivity = 0.,
        _ => panic!("test field {field:?}: expected a required catalog property"),
    }
}
#[test]
fn visual_catalog_ranges_refuse_invisible_or_out_of_range_assets() {
    let (physical, visual) = small_catalog_records();
    for (metalness, roughness, opacity) in [
        (1.1, 0.5, 1.),
        (0.5, -0.1, 1.),
        (0.5, 0.5, 0.),
        (0.5, 0.5, 1.1),
    ] {
        let mut invalid = visual[0].info();
        invalid.base.metalness = metalness;
        invalid.specular.roughness = roughness;
        invalid.geometry.opacity = opacity;
        let refusal = BuiltinAssetCatalog::from_records(
            physical.clone(),
            vec![
                AppearanceAsset::from_info(invalid).unwrap(),
                visual[1].clone(),
            ],
        )
        .unwrap_err();
        assert!(refusal.contains("[0,1]"));
    }
}
#[test]
fn directional_constants_require_their_symmetry_and_complete_positive_moduli() {
    let (physical, visual) = small_catalog_records();
    let mut invalid = physical[0].info();
    invalid.anisotropic.e1 = 20.;
    assert!(
        BuiltinAssetCatalog::from_records(
            vec![MaterialAsset::from_info(invalid.clone()).unwrap()],
            visual.clone()
        )
        .unwrap_err()
        .contains("symmetry")
    );
    invalid.isotropy_class = types::ORTHOTROPIC.0.into_owned();
    assert!(
        BuiltinAssetCatalog::from_records(vec![MaterialAsset::from_info(invalid).unwrap()], visual)
            .unwrap_err()
            .contains("e2")
    );
}
#[test]
fn malformed_magnetic_constitutive_records_are_refused() {
    let (physical, visual) = small_catalog_records();
    for (class, permeability, remanence, coercivity, saturation, expected) in [
        ("", 1., 0., 0., 0., "class"),
        (
            types::HARD_MAGNETIC.0.as_ref(),
            0.,
            0.,
            0.,
            0.,
            "relativePermeability",
        ),
        (types::HARD_MAGNETIC.0.as_ref(), 1., 0., 1., 0., "remanence"),
        (
            types::HARD_MAGNETIC.0.as_ref(),
            1.,
            1.,
            0.,
            0.,
            "coercivity",
        ),
        (
            types::SOFT_MAGNETIC.0.as_ref(),
            1.,
            0.,
            0.,
            0.,
            "saturationFluxDensity",
        ),
        ("alien", 1., 0., 0., 0., "class"),
    ] {
        let mut invalid = physical[0].info();
        invalid.magnetic = types::Magnetic {
            class: types::MagneticClass(class.to_owned().into()),
            relative_permeability: permeability,
            remanence,
            coercivity,
            saturation_flux_density: saturation,
            ..Default::default()
        };
        let refusal = BuiltinAssetCatalog::from_records(
            vec![MaterialAsset::from_info(invalid).unwrap()],
            visual.clone(),
        )
        .unwrap_err();
        assert!(refusal.contains(expected), "{refusal}");
    }
}
