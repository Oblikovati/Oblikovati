// SPDX-License-Identifier: GPL-2.0-only
use super::{MaterialAsset, require_positive_properties};
use oblikovati_api::{
    contract::Material,
    types::{HARD_MAGNETIC, SOFT_MAGNETIC},
};
pub(super) fn require_catalog_magnetic(asset: &MaterialAsset) -> Result<(), String> {
    let magnetic = asset.magnetic();
    if magnetic.class.0.is_empty() {
        return require_unclassified_magnetics(asset);
    }
    require_positive_properties(
        &asset.id(),
        &[("relativePermeability", magnetic.relative_permeability)],
    )?;
    require_magnetic_class_properties(asset)
}
fn require_magnetic_class_properties(asset: &MaterialAsset) -> Result<(), String> {
    let magnetic = asset.magnetic();
    if magnetic.class == HARD_MAGNETIC {
        return require_hard_magnetic_properties(asset);
    }
    if magnetic.class == SOFT_MAGNETIC {
        return require_positive_properties(
            &asset.id(),
            &[("saturationFluxDensity", magnetic.saturation_flux_density)],
        );
    }
    Err(format!(
        "catalog material {:?} magnetic class {:?}: expected {} or {}",
        asset.id(),
        magnetic.class,
        SOFT_MAGNETIC.0,
        HARD_MAGNETIC.0
    ))
}
fn require_hard_magnetic_properties(asset: &MaterialAsset) -> Result<(), String> {
    let magnetic = asset.magnetic();
    require_positive_properties(
        &asset.id(),
        &[
            ("remanence", magnetic.remanence),
            ("coercivity", magnetic.coercivity),
        ],
    )
}
fn require_unclassified_magnetics(asset: &MaterialAsset) -> Result<(), String> {
    let magnetic = asset.magnetic();
    if [
        magnetic.relative_permeability,
        magnetic.remanence,
        magnetic.coercivity,
    ]
    .iter()
    .any(|value| *value != 0.)
    {
        return Err(format!(
            "catalog material {:?} magnetic {magnetic:?}: expected a magnetic class for constitutive values",
            asset.id()
        ));
    }
    Ok(())
}
