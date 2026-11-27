// SPDX-License-Identifier: GPL-2.0-only
//! Authored concrete appearance settings.
use super::{AppearanceHeightMap, AppearanceTint};

/// The Concrete editor recipe carried with a document appearance.
#[derive(Clone, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct ConcreteAppearance {
    pub sealant: ConcreteSealant,
    pub finish: ConcreteFinish,
    pub weathering: bool,
    pub tint: AppearanceTint,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub stamped_image: Option<AppearanceHeightMap>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub weathering_image: Option<AppearanceHeightMap>,
}

impl Default for ConcreteAppearance {
    fn default() -> Self {
        Self {
            sealant: Default::default(),
            finish: Default::default(),
            weathering: true,
            tint: Default::default(),
            stamped_image: None,
            weathering_image: None,
        }
    }
}

impl ConcreteAppearance {
    pub(super) fn validate(&self) -> Result<(), String> {
        for map in self.stamped_image.iter().chain(&self.weathering_image) {
            map.validate()?;
        }
        Ok(())
    }

    /// Compute roughness and highlight strength on the shared scale.
    /// For example, `ConcreteAppearance::default().surface()` gives an unsealed finish.
    pub fn surface(&self) -> [u8; 2] {
        let roughness = match self.finish {
            ConcreteFinish::BroomStraight | ConcreteFinish::BroomCurved => 224,
            ConcreteFinish::Smooth => 164,
            ConcreteFinish::Polished => 97,
            ConcreteFinish::Stamped => 207,
        };
        match self.sealant {
            ConcreteSealant::None => [roughness, 64],
            ConcreteSealant::Epoxy => [roughness / 3, 204],
            ConcreteSealant::Acrylic => [(u16::from(roughness) * 3 / 5) as u8, 147],
        }
    }
}

/// Concrete surface coating.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum ConcreteSealant {
    #[default]
    None,
    Epoxy,
    Acrylic,
}

/// Concrete surface tooling.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum ConcreteFinish {
    #[default]
    BroomStraight,
    BroomCurved,
    Smooth,
    Polished,
    Stamped,
}
