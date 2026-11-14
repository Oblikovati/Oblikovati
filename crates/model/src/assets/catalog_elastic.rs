// SPDX-License-Identifier: GPL-2.0-only
use super::{MaterialAsset, require_positive_properties};
use oblikovati_api::{contract::Material, types::ISOTROPIC};
pub(super) fn require_catalog_elastic(asset: &MaterialAsset) -> Result<(), String> {
    let elastic = asset.anisotropic();
    if asset.isotropy_class() != ISOTROPIC {
        return require_directional_moduli(asset);
    }
    if [elastic.e1, elastic.e2, elastic.e3]
        .iter()
        .any(|value| *value != 0.)
    {
        return Err(format!(
            "catalog material {:?} elastic {elastic:?}: expected an anisotropic symmetry tag for directional moduli",
            asset.id()
        ));
    }
    Ok(())
}

fn require_directional_moduli(asset: &MaterialAsset) -> Result<(), String> {
    let elastic = asset.anisotropic();
    require_positive_properties(
        &asset.id(),
        &[
            ("e1", elastic.e1),
            ("e2", elastic.e2),
            ("e3", elastic.e3),
            ("g12", elastic.g12),
            ("g23", elastic.g23),
            ("g13", elastic.g13),
        ],
    )
}
