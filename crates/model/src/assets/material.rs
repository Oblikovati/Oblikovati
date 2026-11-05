// SPDX-License-Identifier: GPL-2.0-only
use super::{AssetIdentity, validation::validate_asset_record};
use oblikovati_api::{contract::Material, types, wire::MaterialInfo};
#[derive(Clone, Debug, PartialEq)]
pub struct MaterialAsset {
    record: MaterialInfo,
}
impl MaterialAsset {
    /// Retain every API-owned property group; example: `MaterialAsset::from_info(info)?`.
    pub fn from_info(record: MaterialInfo) -> Result<Self, String> {
        validate_asset_record(&record.id, &record.source, &record)?;
        validate_material_properties(&record)?;
        Ok(Self { record })
    }
    asset_snapshot_methods!(MaterialInfo);
}
impl Material for MaterialAsset {
    fn id(&self) -> String {
        self.record.id.clone()
    }
    fn display_name(&self) -> String {
        self.record.display_name.clone()
    }
    fn appearance_id(&self) -> String {
        self.record.appearance_id.clone()
    }
    fn source(&self) -> types::AssetSource {
        types::AssetSource(self.record.source.clone().into())
    }
    fn anisotropic(&self) -> types::AnisotropicElastic {
        self.record.anisotropic.clone()
    }
    fn electrical(&self) -> types::Electrical {
        self.record.electrical.clone()
    }
    fn magnetic(&self) -> types::Magnetic {
        self.record.magnetic.clone()
    }
    fn mechanical(&self) -> types::Mechanical {
        self.record.mechanical.clone()
    }
    fn thermal(&self) -> types::Thermal {
        self.record.thermal.clone()
    }
    fn density(&self) -> f64 {
        self.record.density
    }
    fn isotropy_class(&self) -> types::IsotropyClass {
        if self.record.isotropy_class.is_empty() {
            return types::ISOTROPIC;
        }
        types::IsotropyClass(self.record.isotropy_class.clone().into())
    }
}
impl AssetIdentity for MaterialAsset {
    fn asset_id(&self) -> &str {
        &self.record.id
    }
}

fn validate_material_properties(record: &MaterialInfo) -> Result<(), String> {
    if !record.density.is_finite() || record.density < 0. {
        return Err(format!(
            "material {:?} density {}: expected finite nonnegative g/cm³",
            record.id, record.density
        ));
    }
    validate_material_symmetry(&record.id, &record.isotropy_class)?;
    Ok(())
}

fn validate_material_symmetry(id: &str, symmetry: &str) -> Result<(), String> {
    if ![
        "",
        types::ISOTROPIC.0.as_ref(),
        types::ORTHOTROPIC.0.as_ref(),
        types::TRANSVERSELY_ISOTROPIC.0.as_ref(),
    ]
    .contains(&symmetry)
    {
        return Err(format!(
            "material {:?} isotropy {:?}: expected isotropic, orthotropic, or transversely-isotropic",
            id, symmetry
        ));
    }
    Ok(())
}
