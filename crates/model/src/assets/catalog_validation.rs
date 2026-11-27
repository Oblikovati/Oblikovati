// SPDX-License-Identifier: GPL-2.0-only
use super::{AppearanceAsset, MaterialAsset};
use oblikovati_api::{
    contract::{Appearance, Material},
    types::{ASSET_BUILTIN, AssetSource},
};
#[path = "catalog_elastic.rs"]
mod elastic;
#[path = "catalog_magnetic.rs"]
mod magnetic;
pub(super) fn validate_builtin_material(asset: &MaterialAsset) -> Result<(), String> {
    require_builtin_identity(&asset.id(), &asset.display_name(), asset.source())?;
    require_positive_properties(&asset.id(), &physical_catalog_values(asset))?;
    elastic::require_catalog_elastic(asset)?;
    magnetic::require_catalog_magnetic(asset)
}
pub(super) fn validate_builtin_appearance(asset: &AppearanceAsset) -> Result<(), String> {
    require_builtin_identity(&asset.id(), &asset.display_name(), asset.source())?;
    for (field, value) in [
        ("base.metalness", asset.base().metalness),
        ("specular.roughness", asset.specular().roughness),
        ("geometry.opacity", asset.geometry().opacity),
    ] {
        if !(0. ..=1.).contains(&value) || (field == "geometry.opacity" && value == 0.) {
            return Err(format!(
                "catalog appearance {:?} {field}={value}: expected [0,1] with positive opacity",
                asset.id()
            ));
        }
    }
    Ok(())
}
fn require_builtin_identity(id: &str, name: &str, source: AssetSource) -> Result<(), String> {
    if source != ASSET_BUILTIN || name.is_empty() || name.contains('\0') {
        return Err(format!(
            "catalog asset {id:?}, name {name:?}, source {source:?}: expected a named builtin asset without NUL"
        ));
    }
    Ok(())
}
fn physical_catalog_values(asset: &MaterialAsset) -> Vec<(&'static str, f64)> {
    let mut values = vec![("density", asset.density())];
    values.extend(mechanical_catalog_values(asset));
    let thermal = asset.thermal();
    values.extend([
        ("conductivity", thermal.conductivity),
        ("expansionCoeff", thermal.expansion_coeff),
        ("specificHeat", thermal.specific_heat),
    ]);
    let electrical = asset.electrical();
    values.extend([
        ("resistivity", electrical.resistivity),
        ("relativePermittivity", electrical.relative_permittivity),
    ]);
    values
}
fn mechanical_catalog_values(asset: &MaterialAsset) -> [(&'static str, f64); 4] {
    let mechanical = asset.mechanical();
    [
        ("youngsModulus", mechanical.youngs_modulus),
        ("poissonsRatio", mechanical.poissons_ratio),
        ("yieldStrength", mechanical.yield_strength),
        (
            "ultimateTensileStrength",
            mechanical.ultimate_tensile_strength,
        ),
    ]
}
pub(super) fn require_positive_properties(
    id: &str,
    properties: &[(&str, f64)],
) -> Result<(), String> {
    for (field, value) in properties {
        if *value <= 0. {
            return Err(format!(
                "catalog material {id:?} {field}={value}: expected a positive property value"
            ));
        }
    }
    Ok(())
}
