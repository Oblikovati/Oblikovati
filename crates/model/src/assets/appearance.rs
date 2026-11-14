// SPDX-License-Identifier: GPL-2.0-only
use super::{AssetIdentity, validation::validate_asset_record};
use oblikovati_api::{contract::Appearance, types, wire::AppearanceInfo};
#[derive(Clone, Debug, PartialEq)]
pub struct AppearanceAsset {
    record: AppearanceInfo,
}
impl AppearanceAsset {
    /// Retain every API-owned property group; example: `AppearanceAsset::from_info(info)?`.
    pub fn from_info(record: AppearanceInfo) -> Result<Self, String> {
        validate_asset_record(&record.id, &record.source, &record)?;
        Ok(Self { record })
    }
    asset_snapshot_methods!(AppearanceInfo);
}
impl Appearance for AppearanceAsset {
    fn id(&self) -> String {
        self.record.id.clone()
    }
    fn display_name(&self) -> String {
        self.record.display_name.clone()
    }
    fn source(&self) -> types::AssetSource {
        types::AssetSource(self.record.source.clone().into())
    }
    fn base(&self) -> types::OpenPBRBase {
        self.record.base.clone()
    }
    fn coat(&self) -> types::OpenPBRCoat {
        self.record.coat.clone()
    }
    fn emission(&self) -> types::OpenPBREmission {
        self.record.emission.clone()
    }
    fn fuzz(&self) -> types::OpenPBRFuzz {
        self.record.fuzz.clone()
    }
    fn geometry(&self) -> types::OpenPBRGeometry {
        self.record.geometry.clone()
    }
    fn specular(&self) -> types::OpenPBRSpecular {
        self.record.specular.clone()
    }
    fn subsurface(&self) -> types::OpenPBRSubsurface {
        self.record.subsurface.clone()
    }
    fn thin_film(&self) -> types::OpenPBRThinFilm {
        self.record.thin_film.clone()
    }
    fn transmission(&self) -> types::OpenPBRTransmission {
        self.record.transmission.clone()
    }
}
impl AssetIdentity for AppearanceAsset {
    fn asset_id(&self) -> &str {
        &self.record.id
    }
}
