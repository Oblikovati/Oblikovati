// SPDX-License-Identifier: GPL-2.0-only
//! Authored ceramic controls stored with a document appearance.
use super::{AppearanceBump, AppearanceTint};

/// The ceramic recipe attached to an appearance's common surface data.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct CeramicAppearance {
    pub kind: CeramicKind,
    pub finish: CeramicFinish,
    pub finish_bumps: AppearanceBump,
    pub relief: AppearanceBump,
    pub tint: AppearanceTint,
}

impl CeramicAppearance {
    pub(super) fn validate(self) -> Result<(), String> {
        if self.finish_bumps.amount > 100 || self.relief.amount > 100 {
            return Err(format!(
                "ceramic bump settings {self:?}: expected percentages from 0 to 100"
            ));
        }
        Ok(())
    }
}

/// The type selected in the Ceramic section.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum CeramicKind {
    Ceramic,
    #[default]
    Porcelain,
}

/// The available ceramic surface finishes.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum CeramicFinish {
    #[default]
    Glazed,
    Satin,
    Matte,
}

impl CeramicFinish {
    /// Return roughness and highlight strength on the shared 0–255 scale.
    /// For example, a Matte finish uses `CeramicFinish::Matte.surface()`.
    pub fn surface(self) -> [u8; 2] {
        match self {
            Self::Glazed => [32, 235],
            Self::Satin => [113, 170],
            Self::Matte => [210, 65],
        }
    }
}
